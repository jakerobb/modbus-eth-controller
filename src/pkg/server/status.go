package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/jakerobb/modbus-eth-controller/pkg/modbus"
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
	relayStates, err := modbus.GetStatus(r.Context(), addr)
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read relay states: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	err = enc.Encode(relayStates)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to encode response: %v\n", err)
	}
}
