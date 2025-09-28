package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/swaggo/http-swagger"

	_ "github.com/jakerobb/modbus-eth-controller/docs"
	"github.com/jakerobb/modbus-eth-controller/pkg/server/registry"
	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

type Server struct {
	ProgramDir  string
	Registry    *registry.Registry
	AllowOrigin string
	Logger      *slog.Logger
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

	logger := slog.Default().With("component", "server")

	server := &Server{
		ProgramDir:  programDir,
		Registry:    registry.NewRegistry(),
		AllowOrigin: allowOrigin,
		Logger:      logger,
	}

	server.Registry.LoadProgramsFromDir(util.WithLogger(context.Background(), logger), programDir)
	return server
}

func (server *Server) handle(path string, h http.HandlerFunc, methods ...string) {
	handler := server.wrapWithLogging(h)
	handler = server.wrapWithCors(handler, methods...)
	http.Handle(path, handler)
}

func (server *Server) wrapWithLogging(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		reqLogger := slog.Default().With(
			"req_id", reqID,
			"client_ip", r.RemoteAddr,
			"method", r.Method,
			"path", r.URL.Path,
			"query_string", r.URL.RawQuery,
		)
		ctx := util.WithLogger(r.Context(), reqLogger)
		reqLogger.Info("Received request")
		for name, vals := range r.Header {
			for _, v := range vals {
				reqLogger.Info("header", "name", name, "value", v)
			}
		}
		reqLogger.Info("remote addr", "addr", r.RemoteAddr)
		h.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (server *Server) wrapWithCors(h http.HandlerFunc, methods ...string) http.HandlerFunc {
	allowedMethods := append([]string{"OPTIONS"}, methods...)

	return func(w http.ResponseWriter, r *http.Request) {
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
			server.RespondWithError(r.Context(), w, http.StatusMethodNotAllowed, fmt.Sprintf("%s method not allowed", r.Method))
			return
		}

		h.ServeHTTP(w, r)
	}
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

	server.handle("/run", server.handleRun, "POST")
	server.handle("/programs", server.handlePrograms, "GET")
	server.handle("/status", server.handleStatus, "GET")
	server.handle("/", http.FileServer(http.FS(staticContent)).ServeHTTP, "GET")
	server.handle("/swagger/", httpSwagger.WrapHandler.ServeHTTP, "GET")

	server.Logger.Info("Starting server",
		"address", listenAddr,
		"port", listenPort,
	)
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", listenAddr, listenPort), nil)
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func (server *Server) RespondWithError(ctx context.Context, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Status:  status,
		Message: message,
	})
	if err != nil {
		util.GetLogger(ctx).Error("Failed to encode error response", "error", err)
	}
}
