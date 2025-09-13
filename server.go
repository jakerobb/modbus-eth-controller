package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Server struct {
	ProgramDir      string
	programRegistry map[string]*Program
	registryMutex   sync.RWMutex
}

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

var errorStatus = "error"
var successStatus = "success"

type ProgramResult struct {
	Slug                string     `json:"slug"`
	Program             *Program   `json:"program"`
	Status              *string    `json:"status,omitempty"`
	StartTime           *time.Time `json:"startTime,omitempty"`
	ExecutionTimeMillis *int64     `json:"executionTimeMillis,omitempty"`
	Error               *string    `json:"error,omitempty"`
}

func InitServer() *Server {
	programDir := os.Getenv("MODBUS_PROGRAM_DIR")
	if programDir == "" {
		programDir = "/etc/modbus"
	}

	server := &Server{
		ProgramDir:      programDir,
		programRegistry: make(map[string]*Program),
		registryMutex:   sync.RWMutex{},
	}

	server.loadProgramsFromDir(programDir)
	return server
}

func (server *Server) loadProgramsFromDir(dir string) {
	files, err := os.ReadDir(server.ProgramDir)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to read program directory %s: %v. No programs will be loaded.\n", server.ProgramDir, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		fullPath := filepath.Join(dir, file.Name())
		program, err := ParseProgramFromFile(fullPath)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to parse program from file %s: %v\n", fullPath, err)
			continue
		}

		_, exists := server.programRegistry[program.Slug]
		if exists {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: duplicate program slug '%s' found in file %s, skipping\n", program.Slug, fullPath)
			continue
		}

		server.programRegistry[program.Slug] = program

		fmt.Printf("Loaded program '%s' from %s\n", program.Slug, fullPath)
	}
	fmt.Printf("Total programs loaded: %d\n", len(server.programRegistry))
}

func (server *Server) Start() {
	http.HandleFunc("/run", server.handleRun)
	http.HandleFunc("/programs", server.handlePrograms)

	fmt.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

func (server *Server) handlePrograms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		server.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(server.programRegistry)
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to encode programs: %v", err))
	}
}

func (server *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		server.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var err error
	programs := make([]*Program, 0)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		server.RespondWithError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer CloseQuietly(r.Body)

	if len(body) > 0 {
		program, err := ParseProgram(body)
		if err != nil {
			server.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse program: %v", err))
			return
		}
		program.Slug = "[ad-hoc]"
		programs = append(programs, program)
	}

	query := r.URL.Query()
	slugs, slugsProvided := query["program"]
	if slugsProvided {
		for _, slug := range slugs {
			server.registryMutex.RLock()
			program, exists := server.programRegistry[slug]
			server.registryMutex.RUnlock()
			if !exists {
				program, err = server.LoadNewProgramFromDisk(slug)
				if program == nil {
					server.RespondWithError(w, http.StatusNotFound, fmt.Sprintf("Program '%s' not found", slug))
					return
				}
			}

			program, err = server.ReloadProgramFromDiskIfNewer(program)
			if err != nil {
				server.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reload program: %v", err))
				return
			}

			if program == nil {
				server.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Internal error: program for '%s' is nil", slug))
				return
			}

			programs = append(programs, program)
		}
	}

	if len(programs) == 0 {
		server.RespondWithError(w, http.StatusInternalServerError, "Internal error: no programs to run")
		return
	}

	results := make([]ProgramResult, 0)
	for _, program := range programs {
		result := ProgramResult{
			Slug:    program.Slug,
			Program: program,
			Status:  nil,
			Error:   nil,
		}
		startTime := time.Now()
		err := program.Run()
		endTime := time.Now()
		result.Slug = program.Slug
		result.StartTime = &startTime
		result.ExecutionTimeMillis = new(int64)
		*result.ExecutionTimeMillis = endTime.Sub(startTime).Milliseconds()
		if err != nil {
			result.Status = &errorStatus
			result.Error = new(string)
			*result.Error = err.Error()
		} else {
			result.Status = &successStatus
		}
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	err = enc.Encode(results)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to encode response: %v\n", err)
	}
}

func (server *Server) RespondWithError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Status:  status,
		Message: message,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to encode error response: %v\n", err)
	}
}

func (server *Server) LoadNewProgramFromDisk(slug string) (*Program, error) {
	server.registryMutex.Lock()
	_, exists := server.programRegistry[slug]
	server.registryMutex.Unlock()
	if exists {
		return nil, fmt.Errorf("program '%s' already exists", slug)
	}

	files, err := os.ReadDir(server.ProgramDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read program directory %s: %v", server.ProgramDir, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") || SlugifyFilename(file.Name()) != slug {
			continue
		}
		fullPath := filepath.Join(server.ProgramDir, file.Name())
		program, err := ParseProgramFromFile(fullPath)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to parse program from file %s: %v\n", fullPath, err)
			continue
		}

		_, exists := server.programRegistry[program.Slug]
		if exists {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: duplicate program slug '%s' found in file %s, skipping\n", program.Slug, fullPath)
			continue
		}

		fmt.Printf("Loaded new program '%s' from %s\n", program.Slug, fullPath)
		server.registryMutex.Lock()
		server.programRegistry[program.Slug] = program
		server.registryMutex.Unlock()
		return program, nil
	}
	return nil, nil
}

func (server *Server) ReloadProgramFromDiskIfNewer(program *Program) (*Program, error) {
	path := program.Path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat program file: %v", err)
	}
	modTime := info.ModTime()
	if modTime.After(*program.LastModified) {
		newProgram, err := ParseProgramFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to reload program: %v", err)
		}
		server.registryMutex.Lock()
		oldMod := program.LastModified
		server.programRegistry[program.Slug] = newProgram
		server.registryMutex.Unlock()
		fmt.Printf("Reloaded program '%s': old mod time %v, new mod time %v\n", program.Slug, oldMod, newProgram.LastModified)
		return newProgram, nil
	}
	return program, nil
}
