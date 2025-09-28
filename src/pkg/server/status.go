package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jakerobb/modbus-eth-controller/pkg/modbus"
	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

// handleStatus godoc
// @Summary      Get status of all relays
// @Description  Returns the current state of all relays for the specified Modbus device
// @Tags         status
// @Produce      json
// @Param        address query string true "Modbus device IP or hostname and port number"
// @Success      200 {object} modbus.CoilStates
// @Failure      500 {object} server.ErrorResponse
// @Router       /status [get]
func (server *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("address")
	ctx := r.Context()
	query := r.URL.Query()
	debugParam := query.Get("debug")
	if debugParam == "true" {
		ctx = context.WithValue(ctx, "debug", true)
	}
	relayStates, err := modbus.GetStatus(ctx, addr)
	if err != nil {
		server.RespondWithError(ctx, w, http.StatusInternalServerError, fmt.Sprintf("Failed to read relay states: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	err = enc.Encode(relayStates)
	if err != nil {
		logger := util.GetLogger(ctx)
		logger.Error("Failed to encode response", "error", err)
	}
}
