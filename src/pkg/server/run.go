package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jakerobb/modbus-eth-controller/pkg/api"
	"github.com/jakerobb/modbus-eth-controller/pkg/modbus"
	"github.com/jakerobb/modbus-eth-controller/pkg/util"
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
	var err error
	programs := make([]*api.Program, 0)

	ctx := r.Context()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		server.RespondWithError(ctx, w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer util.CloseQuietly(r.Body)

	if len(body) > 0 {
		program, err := api.ParseProgram(body)
		if err != nil {
			server.RespondWithError(ctx, w, http.StatusBadRequest, fmt.Sprintf("Failed to parse program: %v", err))
			return
		}
		program.Slug = "[ad-hoc]"
		programs = append(programs, program)
	}

	query := r.URL.Query()
	debugParam := query.Get("debug")
	if debugParam == "true" {
		ctx = context.WithValue(ctx, "debug", true)
	}

	slugs, slugsProvided := query["program"]
	if slugsProvided {
		err, status, returnedPrograms := server.getNamedPrograms(ctx, slugs)
		if err != nil {
			server.RespondWithError(ctx, w, status, err.Error())
			return
		}
		programs = append(programs, returnedPrograms...)
	}

	if len(programs) == 0 {
		server.RespondWithError(ctx, w, http.StatusInternalServerError, "Internal error: no programs to run")
		return
	}

	results, servers := server.runPrograms(ctx, programs)

	relayStatesByServer := server.collectRelayStates(ctx, servers)

	runResponse := RunResponse{
		Results: results,
		Status:  relayStatesByServer,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err = enc.Encode(runResponse)
	if err != nil {
		logger := util.GetLogger(ctx)
		logger.Error("Failed to encode response", "error", err)
	}
}

func (server *Server) getNamedPrograms(ctx context.Context, slugs []string) (error, int, []*api.Program) {
	programs := make([]*api.Program, 0)
	for _, slug := range slugs {
		program, exists := server.Registry.GetProgram(slug)
		if !exists {
			_, err := server.Registry.LoadNewProgramFromDisk(ctx, slug, server.ProgramDir)
			if err != nil {
				err = fmt.Errorf("failed to load program: %w", err)
				return err, http.StatusNotFound, nil
			}
		}

		program, err := server.Registry.ReloadProgramFromDiskIfNewer(ctx, program)
		if err != nil {
			err = fmt.Errorf("failed to reload program: %w", err)
			return err, http.StatusInternalServerError, nil
		}

		if program == nil {
			err = fmt.Errorf("internal error: program for '%s' is nil", slug)
			return err, http.StatusInternalServerError, nil
		}

		programs = append(programs, program)
	}
	return nil, 0, programs
}

func (server *Server) runPrograms(ctx context.Context, programs []*api.Program) ([]ProgramResult, []string) {
	servers := util.NewSet()
	results := make([]ProgramResult, 0)
	for _, program := range programs {
		result := ProgramResult{
			Slug:    program.Slug,
			Program: program,
			Status:  nil,
			Error:   nil,
		}
		startTime := time.Now()
		programCtx := ctx
		if program.Debug {
			programCtx = context.WithValue(ctx, "debug", true)
		}
		err := program.Run(programCtx)
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
			servers.Add(program.Address)
		}
		results = append(results, result)
	}
	return results, servers.ToArray()
}

func (server *Server) collectRelayStates(ctx context.Context, servers []string) map[string]*modbus.CoilStates {
	util.LogDebug(ctx, "Programs complete. Collecting status.", "servers", servers)

	relayStatesByServer := make(map[string]*modbus.CoilStates)
	for _, serverAddr := range servers {
		if len(serverAddr) == 0 {
			continue
		}
		relayStates, err := modbus.GetStatus(ctx, serverAddr)
		if err == nil {
			relayStatesByServer[serverAddr] = relayStates
		}
	}
	return relayStatesByServer
}
