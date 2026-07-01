package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/errtext"
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
	if len(parts) == 5 && parts[4] == "invalid-issues" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		server.listDataTaskInvalidIssues(w, r, parts[3])
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
	if len(parts) == 5 && parts[4] == "repair-invalid-issues" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		server.repairDataTaskInvalidIssues(w, r, parts[3])
		return
	}
	if len(parts) == 5 && parts[4] == "repair-gap" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		server.repairDataTaskGap(w, r, parts[3])
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
		writeJSON(w, http.StatusOK, sanitizeDataSyncTasks(tasks))
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
		if !server.requireActiveMarketInstrument(w, r, request.Exchange, request.Symbol) {
			return
		}
		task, err := server.repository.CreateDataSyncTask(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, sanitizeDataSyncTask(task))
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
		if server.writeDataSyncTaskMarketInstrumentCommandError(w, r, command.id, err) {
			return
		}
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sanitizeDataSyncTask(task))
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
		if server.writeDataSyncTaskMarketInstrumentCommandError(w, r, id, err) {
			return
		}
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sanitizeDataSyncTask(task))
}

func (server *Server) writeDataSyncTaskMarketInstrumentCommandError(
	w http.ResponseWriter,
	r *http.Request,
	id string,
	err error,
) bool {
	code, ok := data.DomainErrorCode(err)
	if !ok || code != data.ErrorCodeMarketInstrumentNotActive {
		return false
	}
	message, ok := server.dataSyncTaskMarketInstrumentMessage(r, id)
	if !ok {
		message = marketInstrumentNotActiveMessage
	}
	writeAPIError(w, http.StatusBadRequest, apiErrorMarketInstrumentNotActive, message)
	return true
}

func (server *Server) dataSyncTaskMarketInstrumentMessage(r *http.Request, id string) (string, bool) {
	tasks, err := server.repository.ListDataSyncTasks(r.Context())
	if err != nil {
		return "", false
	}
	for _, task := range tasks {
		if task.ID != id {
			continue
		}
		switch task.MarketStatus {
		case data.DataSyncMarketStatusInactive:
			return marketInstrumentInactiveMessage, true
		case data.DataSyncMarketStatusMissing:
			return marketInstrumentMissingMessage, true
		default:
			return "", false
		}
	}
	return "", false
}

func (server *Server) listDataTaskGaps(w http.ResponseWriter, r *http.Request, id string) {
	gaps, err := server.repository.ListDataSyncTaskGaps(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gaps)
}

func (server *Server) listDataTaskInvalidIssues(w http.ResponseWriter, r *http.Request, id string) {
	query, err := parseDataSyncInvalidIssueQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	issues, err := server.repository.ListDataSyncTaskInvalidIssues(r.Context(), id, query)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, issues)
}

func (server *Server) repairDataTaskGaps(w http.ResponseWriter, r *http.Request, id string) {
	result, err := server.repository.RepairDataSyncTaskGaps(r.Context(), id)
	if err != nil {
		if server.writeDataSyncTaskMarketInstrumentCommandError(w, r, id, err) {
			return
		}
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sanitizeDataSyncGapRepairResult(result))
}

func (server *Server) repairDataTaskInvalidIssues(w http.ResponseWriter, r *http.Request, id string) {
	var request data.RepairDataSyncInvalidIssuesRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateRepairDataSyncInvalidIssuesRequest(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := server.repository.RepairDataSyncTaskInvalidIssues(r.Context(), id, request)
	if err != nil {
		if server.writeDataSyncTaskMarketInstrumentCommandError(w, r, id, err) {
			return
		}
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sanitizeDataSyncGapRepairResult(result))
}

func (server *Server) repairDataTaskGap(w http.ResponseWriter, r *http.Request, id string) {
	var request data.RepairDataSyncTaskGapRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.From.IsZero() || request.To.IsZero() || !request.From.Before(request.To) {
		writeError(w, http.StatusBadRequest, "from and to are required and from must be before to")
		return
	}
	interval, ok, err := server.dataSyncTaskInterval(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if ok {
		from := request.From.UTC()
		to := request.To.UTC()
		if err := data.ValidateDataSyncTaskWindow(interval, &from, &to); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		request.From = from
		request.To = to
	}
	result, err := server.repository.RepairDataSyncTaskGap(r.Context(), id, request)
	if err != nil {
		if server.writeDataSyncTaskMarketInstrumentCommandError(w, r, id, err) {
			return
		}
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sanitizeDataSyncGapRepairResult(result))
}

func (server *Server) dataSyncTaskInterval(ctx context.Context, id string) (string, bool, error) {
	tasks, err := server.repository.ListDataSyncTasks(ctx)
	if err != nil {
		return "", false, err
	}
	for _, task := range tasks {
		if task.ID == id {
			return task.Interval, true, nil
		}
	}
	return "", false, nil
}

func sanitizeDataSyncTasks(tasks []data.DataSyncTask) []data.DataSyncTask {
	result := make([]data.DataSyncTask, len(tasks))
	for index, task := range tasks {
		result[index] = sanitizeDataSyncTask(task)
	}
	return result
}

func sanitizeDataSyncGapRepairResult(result data.DataSyncGapRepairResult) data.DataSyncGapRepairResult {
	for index, task := range result.CreatedTasks {
		result.CreatedTasks[index] = sanitizeDataSyncTask(task)
	}
	return result
}

func sanitizeDataSyncTask(task data.DataSyncTask) data.DataSyncTask {
	task.LastError = sanitizeExternalError(task.LastError)
	task.ExchangeBackoffError = sanitizeExternalError(task.ExchangeBackoffError)
	return task
}

func sanitizeExternalError(value string) string {
	return errtext.ExternalError(value)
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

func parseDataSyncInvalidIssueQuery(r *http.Request) (data.DataSyncInvalidIssueQuery, error) {
	values := r.URL.Query()
	query := data.DataSyncInvalidIssueQuery{
		Limit: data.DefaultDataSyncInvalidIssueLimit,
	}
	if rawLimit := values.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return data.DataSyncInvalidIssueQuery{}, errors.New("limit must be a positive integer")
		}
		if limit > data.MaxDataSyncInvalidIssueLimit {
			return data.DataSyncInvalidIssueQuery{}, fmt.Errorf("limit must be less than or equal to %d", data.MaxDataSyncInvalidIssueLimit)
		}
		query.Limit = limit
	}
	if rawOffset := values.Get("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return data.DataSyncInvalidIssueQuery{}, errors.New("offset must be a non-negative integer")
		}
		query.Offset = offset
	}
	if rawCode := strings.TrimSpace(values.Get("code")); rawCode != "" {
		if !data.IsCandleIssueCode(rawCode) {
			return data.DataSyncInvalidIssueQuery{}, errors.New("code must be a known invalid candle issue")
		}
		query.Code = rawCode
	}
	from, err := parseOptionalTime(values.Get("from"))
	if err != nil {
		return data.DataSyncInvalidIssueQuery{}, fmt.Errorf("from: %w", err)
	}
	to, err := parseOptionalTime(values.Get("to"))
	if err != nil {
		return data.DataSyncInvalidIssueQuery{}, fmt.Errorf("to: %w", err)
	}
	if from != nil && to != nil && from.After(*to) {
		return data.DataSyncInvalidIssueQuery{}, errors.New("from must be before or equal to to")
	}
	query.From = from
	query.To = to
	return data.NormalizeDataSyncInvalidIssueQuery(query), nil
}

func validateRepairDataSyncInvalidIssuesRequest(request data.RepairDataSyncInvalidIssuesRequest) error {
	if request.Code != "" && !data.IsCandleIssueCode(request.Code) {
		return errors.New("code must be a known invalid candle issue")
	}
	if request.From != nil && request.To != nil && request.From.After(*request.To) {
		return errors.New("from must be before or equal to to")
	}
	return nil
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
	if rawCursor := values.Get("cursor"); rawCursor != "" {
		if values.Get("from") != "" || values.Get("to") != "" || values.Get("limit") != "" {
			return data.CandleQuery{}, errors.New("cursor cannot be combined with from, to or limit")
		}
		cursor, err := data.DecodeCandleCursor(rawCursor)
		if err != nil {
			return data.CandleQuery{}, err
		}
		if !cursor.MatchesQuery(query) {
			return data.CandleQuery{}, errors.New("cursor does not match exchange, symbol or interval")
		}
		from := cursor.From
		to := cursor.To
		query.From = &from
		query.To = &to
		query.Limit = cursor.Limit
		if err := data.ValidateCandleQueryRange(query); err != nil {
			return data.CandleQuery{}, err
		}
		return query, nil
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
