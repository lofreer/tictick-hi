package api

import (
	"net/http"
)

func (server *Server) handleStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	parts := pathParts(r.URL.Path)
	switch len(parts) {
	case 2:
		strategies, err := server.strategyRepository.ListStrategies(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, strategies)
	case 3:
		definition, err := server.strategyRepository.GetStrategy(r.Context(), parts[2])
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, definition)
	default:
		writeError(w, http.StatusNotFound, "strategy route not found")
	}
}
