package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	if len(parts) == 3 && parts[2] == "trends" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		server.listOverviewTrends(w, r)
		return
	}
	writeError(w, http.StatusNotFound, "overview route not found")
}

func (server *Server) listOverviewRecentFacts(w http.ResponseWriter, r *http.Request) {
	query, err := parseOverviewRecentFactQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	facts, err := server.repository.ListOverviewRecentFacts(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, facts)
}

func (server *Server) listOverviewTrends(w http.ResponseWriter, r *http.Request) {
	query, err := parseOverviewTrendQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	trends, err := server.repository.ListOverviewTrends(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, trends)
}

func parseOverviewRecentFactQuery(r *http.Request) (data.OverviewRecentFactQuery, error) {
	rawLimit := r.URL.Query().Get("limit")
	query := data.OverviewRecentFactQuery{Limit: data.DefaultOverviewRecentFactLimit}
	if rawLimit == "" {
		return parseOverviewRecentFactSince(r, query)
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit <= 0 {
		return data.OverviewRecentFactQuery{}, fmt.Errorf("limit must be a positive integer")
	}
	if limit > data.MaxOverviewRecentFactLimit {
		return data.OverviewRecentFactQuery{}, fmt.Errorf("limit must be less than or equal to %d", data.MaxOverviewRecentFactLimit)
	}
	query.Limit = limit
	return parseOverviewRecentFactSince(r, query)
}

func parseOverviewRecentFactSince(r *http.Request, query data.OverviewRecentFactQuery) (data.OverviewRecentFactQuery, error) {
	rawSince := r.URL.Query().Get("since")
	if rawSince == "" {
		return query, nil
	}
	since, err := time.Parse(time.RFC3339Nano, rawSince)
	if err != nil {
		return data.OverviewRecentFactQuery{}, fmt.Errorf("since must be an RFC3339 date-time")
	}
	since = since.UTC()
	query.Since = &since
	return query, nil
}

func parseOverviewTrendQuery(r *http.Request) (data.OverviewTrendQuery, error) {
	rawDays := r.URL.Query().Get("days")
	days := data.DefaultOverviewTrendDays
	if rawDays != "" {
		parsed, err := strconv.Atoi(rawDays)
		if err != nil || parsed <= 0 {
			return data.OverviewTrendQuery{}, fmt.Errorf("days must be a positive integer")
		}
		if parsed > data.MaxOverviewTrendDays {
			return data.OverviewTrendQuery{}, fmt.Errorf("days must be less than or equal to %d", data.MaxOverviewTrendDays)
		}
		days = parsed
	}
	to := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	return data.OverviewTrendQuery{
		Days: days,
		From: to.AddDate(0, 0, -days),
		To:   to,
	}, nil
}
