package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	. "github.com/jakerobb/modbus-eth-controller/pkg"
	"github.com/jakerobb/modbus-eth-controller/pkg/api"
	"github.com/jakerobb/modbus-eth-controller/pkg/modbus"
)

type RunStatus string

var errorStatus RunStatus = "error"
var successStatus RunStatus = "success"

type RunResponse struct {
	Results []ProgramResult               `json:"results"`
	Status  map[string]*modbus.CoilStates `json:"status"`
}

type ProgramResult struct {
	Status              *RunStatus   `json:"status" example:"success"`
	Error               *string      `json:"error,omitempty" example:"relay 1 timed out"`
	StartTime           *time.Time   `json:"startTime" example:"2025-01-01T12:00:00Z"`
	ExecutionTimeMillis *int64       `json:"executionTimeMillis" example:"153"`
	Slug                string       `json:"slug" example:"doorbell"`
	Program             *api.Program `json:"program"`
}

// handleRun godoc
// @Summary      Run one or more programs
// @Description  Executes programs in order. You can provide:
// @Description  1. A program in the request body
// @Description  2. Program slug(s) via the `program` query parameter
// @Description  3. Both — the body program runs first, then the slugged programs in order
// @Tags         run
// @Accept       json
// @Produce      json
// @Param        program query []string false "Program slug (repeatable)" collectionFormat(multi)
// @Param        program body api.Program false "Inline program to run"
// @Success      200 {object} server.RunResponse
// @Failure      400 {object} server.ErrorResponse "if the request body is malformed"
// @Failure      404 {object} server.ErrorResponse "if a named program is not found"
// @Failure      500 {object} server.ErrorResponse
// @Router       /run [post]
func (server *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		server.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var err error
	programs := make([]*api.Program, 0)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		server.RespondWithError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer CloseQuietly(r.Body)

	if len(body) > 0 {
		program, err := api.ParseProgram(body)
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
			program, exists := server.Registry.GetProgram(slug)
			if !exists {
				program, err = server.Registry.LoadNewProgramFromDisk(slug, server.ProgramDir)
				if program == nil {
					server.RespondWithError(w, http.StatusNotFound, fmt.Sprintf("Program '%s' not found", slug))
					return
				}
			}

			program, err = server.Registry.ReloadProgramFromDiskIfNewer(program)
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

	servers := make([]string, 0)

	results := make([]ProgramResult, 0)
	for _, program := range programs {
		result := ProgramResult{
			Slug:    program.Slug,
			Program: program,
			Status:  nil,
			Error:   nil,
		}
		servers = append(servers, program.Address)
		startTime := time.Now()
		ctx := context.WithValue(r.Context(), "debug", program.Debug)
		err := program.Run(ctx)
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

	relayStatesByServer := make(map[string]*modbus.CoilStates)
	for _, serverAddr := range servers {
		relayStates, err := modbus.GetStatus(r.Context(), serverAddr)
		if err != nil {
			server.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get status after running programs: %v", err))
		}
		relayStatesByServer[serverAddr] = relayStates
	}

	runResponse := RunResponse{
		Results: results,
		Status:  relayStatesByServer,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	err = enc.Encode(runResponse)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to encode response: %v\n", err)
	}
}
