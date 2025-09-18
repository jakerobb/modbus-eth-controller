package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/swaggo/http-swagger"

	_ "github.com/jakerobb/modbus-eth-controller/docs"
	"github.com/jakerobb/modbus-eth-controller/pkg/server/registry"
)

type Server struct {
	ProgramDir string
	Registry   *registry.Registry
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

	server := &Server{
		ProgramDir: programDir,
		Registry:   registry.NewRegistry(),
	}

	server.Registry.LoadProgramsFromDir(programDir)
	return server
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

	http.HandleFunc("/run", server.handleRun)
	http.HandleFunc("/programs", server.handlePrograms)
	http.HandleFunc("/status", server.handleStatus)
	http.Handle("/", http.FileServer(http.FS(staticContent)))
	http.Handle("/swagger/", httpSwagger.WrapHandler)

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
