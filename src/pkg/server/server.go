package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/swaggo/http-swagger"

	_ "github.com/jakerobb/modbus-eth-controller/docs"
	"github.com/jakerobb/modbus-eth-controller/pkg/server/registry"
)

type Server struct {
	ProgramDir  string
	Registry    *registry.Registry
	AllowOrigin string
}

type ErrorResponse struct {
	Status  int    `json:"status" example:"500"`
	Message string `json:"message" example:"Invalid relay number"`
}

func InitServer() *Server {
	programDir := os.Getenv("MODBUS_PROGRAM_DIR")
	if programDir == "" {
		programDir = "/etc/modbus"
	}

	allowOrigin := os.Getenv("ALLOW_ORIGIN")
	if allowOrigin == "" {
		allowOrigin = "*"
	}

	server := &Server{
		ProgramDir:  programDir,
		Registry:    registry.NewRegistry(),
		AllowOrigin: allowOrigin,
	}

	server.Registry.LoadProgramsFromDir(programDir)
	return server
}

func (server *Server) handleWithCORS(path string, h http.HandlerFunc, methods ...string) {
	allowedMethods := append([]string{"OPTIONS"}, methods...)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", server.AllowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		methodAllowed := false
		for _, m := range methods {
			if r.Method == m {
				methodAllowed = true
				break
			}
		}
		if !methodAllowed {
			server.RespondWithError(w, http.StatusMethodNotAllowed, fmt.Sprintf("%s method not allowed", r.Method))
			return
		}

		h.ServeHTTP(w, r)
	})

	http.Handle(path, handler)
}

func (server *Server) Start() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "0.0.0.0"
	}

	listenPort := os.Getenv("LISTEN_PORT")
	if listenPort == "" {
		listenPort = "8080"
	}

	server.handleWithCORS("/run", server.handleRun, "POST")
	server.handleWithCORS("/programs", server.handlePrograms, "GET")
	server.handleWithCORS("/status", server.handleStatus, "GET")
	server.handleWithCORS("/", http.FileServer(http.FS(staticContent)).ServeHTTP, "GET")
	server.handleWithCORS("/swagger/", httpSwagger.WrapHandler.ServeHTTP, "GET")

	fmt.Printf("Starting server on %s:%s\n", listenAddr, listenPort)
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", listenAddr, listenPort), nil)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
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
