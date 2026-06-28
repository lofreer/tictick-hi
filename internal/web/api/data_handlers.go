package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (server *Server) handleDataTasks(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 3 {
		server.handleTaskCollection(w, r)
		return
	}
	if len(parts) == 4 {
		if r.Method != http.MethodDelete {
			writeMethodNotAllowed(w, http.MethodDelete)
			return
		}
		server.deleteDataTask(w, r, parts[3])
		return
	}
	if len(parts) == 5 && parts[4] == "retry" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		server.retryDataTask(w, r, parts[3])
		return
	}
	if len(parts) == 5 && parts[4] == "gaps" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		server.listDataTaskGaps(w, r, parts[3])
		return
	}
	if len(parts) == 5 && parts[4] == "repair-gaps" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		server.repairDataTaskGaps(w, r, parts[3])
		return
	}
	if len(parts) == 6 && (parts[4] == "sync" || parts[4] == "realtime") {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		server.handleTaskCommand(w, r, taskCommand{
			id:       parts[3],
			category: parts[4],
			action:   parts[5],
		})
		return
	}
	writeError(w, http.StatusNotFound, "data task route not found")
}

func (server *Server) handleTaskCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasks, err := server.repository.ListDataSyncTasks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		var request data.CreateDataSyncTask
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validateCreateTask(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task, err := server.repository.CreateDataSyncTask(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (server *Server) handleTaskCommand(w http.ResponseWriter, r *http.Request, command taskCommand) {
	var (
		task data.DataSyncTask
		err  error
	)

	switch command {
	case taskCommand{id: command.id, category: "sync", action: "start"}:
		task, err = server.repository.SetSyncEnabled(r.Context(), command.id, true)
	case taskCommand{id: command.id, category: "sync", action: "stop"}:
		task, err = server.repository.SetSyncEnabled(r.Context(), command.id, false)
	case taskCommand{id: command.id, category: "realtime", action: "start"}:
		task, err = server.repository.SetRealtimeEnabled(r.Context(), command.id, true)
	case taskCommand{id: command.id, category: "realtime", action: "stop"}:
		task, err = server.repository.SetRealtimeEnabled(r.Context(), command.id, false)
	default:
		writeError(w, http.StatusNotFound, "data task command not found")
		return
	}

	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) deleteDataTask(w http.ResponseWriter, r *http.Request, id string) {
	if err := server.repository.DeleteDataSyncTask(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (server *Server) retryDataTask(w http.ResponseWriter, r *http.Request, id string) {
	task, err := server.repository.RetryDataSyncTask(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) listDataTaskGaps(w http.ResponseWriter, r *http.Request, id string) {
	gaps, err := server.repository.ListDataSyncTaskGaps(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gaps)
}

func (server *Server) repairDataTaskGaps(w http.ResponseWriter, r *http.Request, id string) {
	result, err := server.repository.RepairDataSyncTaskGaps(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleCandles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	query, err := parseCandleQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := server.repository.GetCandles(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type taskCommand struct {
	id       string
	category string
	action   string
}

func parseCandleQuery(r *http.Request) (data.CandleQuery, error) {
	values := r.URL.Query()
	query := data.CandleQuery{
		Exchange: values.Get("exchange"),
		Symbol:   values.Get("symbol"),
		Interval: values.Get("interval"),
		Limit:    data.DefaultCandleLimit,
	}
	if query.Exchange == "" || query.Symbol == "" || query.Interval == "" {
		return data.CandleQuery{}, errors.New("exchange, symbol and interval are required")
	}
	if err := validateExchangeSymbol(query.Exchange, query.Symbol); err != nil {
		return data.CandleQuery{}, err
	}
	if rawLimit := values.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return data.CandleQuery{}, errors.New("limit must be a positive integer")
		}
		if limit > data.MaxCandleLimit {
			return data.CandleQuery{}, fmt.Errorf("limit must be less than or equal to %d", data.MaxCandleLimit)
		}
		query.Limit = limit
	}
	from, err := parseOptionalTime(values.Get("from"))
	if err != nil {
		return data.CandleQuery{}, fmt.Errorf("from: %w", err)
	}
	to, err := parseOptionalTime(values.Get("to"))
	if err != nil {
		return data.CandleQuery{}, fmt.Errorf("to: %w", err)
	}
	query.From = from
	query.To = to
	if err := data.ValidateCandleQueryRange(query); err != nil {
		return data.CandleQuery{}, err
	}
	return query, nil
}
