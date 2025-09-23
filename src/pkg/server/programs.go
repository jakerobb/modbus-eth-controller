package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jakerobb/modbus-eth-controller/pkg/api"
)

// ProgramsBySlugExample is a docs-only type used to present example keys for a
// map[string]api.Program response in Swagger UI. At runtime, the endpoint returns
// a map keyed by slug; the fields below are illustrative examples only.
type ProgramsBySlugExample struct {
	Doorbell api.Program `json:"doorbell"`
}

// handlePrograms godoc
// @Summary      List known programs
// @Description  Returns all available programs keyed by slug
// @Tags         programs
// @Produce      json
// @Success      200 {object} ProgramsBySlugExample
// @Failure      500 {object} server.ErrorResponse
// @Router       /programs [get]
func (server *Server) handlePrograms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(server.Registry.Programs)
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to encode programs: %v", err))
	}
}
