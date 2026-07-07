package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (server *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 3 && parts[2] == "recent-facts" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		server.listOverviewRecentFacts(w, r)
		return
	}
	writeError(w, http.StatusNotFound, "overview route not found")
}

func (server *Server) listOverviewRecentFacts(w http.ResponseWriter, r *http.Request) {
	limit, err := parseOverviewRecentFactLimit(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	facts, err := server.repository.ListOverviewRecentFacts(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, facts)
}

func parseOverviewRecentFactLimit(r *http.Request) (int, error) {
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit == "" {
		return data.DefaultOverviewRecentFactLimit, nil
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}
	if limit > data.MaxOverviewRecentFactLimit {
		return 0, fmt.Errorf("limit must be less than or equal to %d", data.MaxOverviewRecentFactLimit)
	}
	return limit, nil
}
