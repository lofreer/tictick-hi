package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

const maxMarketInstrumentQueryLimit = 50

func (server *Server) handleMarket(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/market/candle-gaps":
		server.handleMarketCandleGaps(w, r)
	case "/api/market/candle-invalid-issues":
		server.handleMarketCandleInvalidIssues(w, r)
	case "/api/market/candle-gaps/repair":
		server.repairMarketCandleGap(w, r)
	case "/api/market/candle-gaps/repair-batch":
		server.repairMarketCandleGaps(w, r)
	case "/api/market/candle-invalid-issues/repair":
		server.repairMarketCandleInvalidIssues(w, r)
	case "/api/market/instruments":
		server.handleMarketInstruments(w, r)
	case "/api/market/instruments/status":
		server.handleMarketInstrumentSyncStatuses(w, r)
	case "/api/market/instruments/sync":
		server.syncMarketInstruments(w, r)
	default:
		writeError(w, http.StatusNotFound, "market route not found")
	}
}

func (server *Server) handleMarketCandleGaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	query, err := parseMarketCandleGapScanQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	scan, err := server.repository.ScanMarketCandleGaps(r.Context(), query)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

func (server *Server) handleMarketCandleInvalidIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	query, err := parseMarketCandleInvalidIssueScanQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	scan, err := server.repository.ScanMarketCandleInvalidIssues(r.Context(), query)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

func (server *Server) repairMarketCandleGap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var request data.RepairMarketCandleGapRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateRepairMarketCandleGapRequest(&request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !server.requireActiveMarketInstrument(w, r, request.Exchange, request.Symbol) {
		return
	}
	result, err := server.repository.RepairMarketCandleGap(r.Context(), request)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sanitizeDataSyncGapRepairResult(result))
}

func (server *Server) repairMarketCandleGaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var request data.RepairMarketCandleGapsRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateRepairMarketCandleGapsRequest(&request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !server.requireActiveMarketInstrument(w, r, request.Exchange, request.Symbol) {
		return
	}
	result, err := server.repository.RepairMarketCandleGaps(r.Context(), request)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sanitizeDataSyncGapRepairResult(result))
}

func (server *Server) repairMarketCandleInvalidIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var request data.RepairMarketCandleInvalidIssuesRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateRepairMarketCandleInvalidIssuesRequest(&request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !server.requireActiveMarketInstrument(w, r, request.Exchange, request.Symbol) {
		return
	}
	result, err := server.repository.RepairMarketCandleInvalidIssues(r.Context(), request)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sanitizeDataSyncGapRepairResult(result))
}

func (server *Server) requireActiveMarketInstrument(
	w http.ResponseWriter,
	r *http.Request,
	exchange string,
	symbol string,
) bool {
	if _, err := server.repository.GetActiveMarketInstrument(r.Context(), exchange, symbol); err != nil {
		if errors.Is(err, data.ErrNotFound) {
			writeAPIError(w, http.StatusBadRequest, apiErrorMarketInstrumentNotActive, "market instrument is not active in catalog")
			return false
		}
		writeStoreError(w, err)
		return false
	}
	return true
}

func (server *Server) handleMarketInstruments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	query, err := parseMarketInstrumentQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	instruments, err := server.repository.ListMarketInstruments(r.Context(), query)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, instruments)
}

func (server *Server) handleMarketInstrumentSyncStatuses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	statuses, err := server.repository.ListMarketInstrumentSyncStatuses(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, statuses)
}

func (server *Server) syncMarketInstruments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	query, err := parseMarketInstrumentSyncQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	client := server.instrumentClients[query.Exchange]
	if client == nil {
		writeAPIError(w, http.StatusBadRequest, apiErrorRequestFailed, "market instrument sync is unavailable for "+query.Exchange)
		return
	}
	attemptedAt := time.Now().UTC()
	instruments, err := client.FetchInstruments(r.Context())
	if err != nil {
		if recordErr := server.repository.RecordMarketInstrumentSyncFailure(
			r.Context(),
			query.Exchange,
			err,
			attemptedAt,
		); recordErr != nil {
			writeStoreError(w, recordErr)
			return
		}
		writeAPIError(w, http.StatusBadRequest, apiErrorRequestFailed, "sync market instruments: "+err.Error())
		return
	}
	result, err := server.repository.ReplaceMarketInstruments(r.Context(), query.Exchange, instruments, time.Now().UTC())
	if err != nil {
		_ = server.repository.RecordMarketInstrumentSyncFailure(
			r.Context(),
			query.Exchange,
			err,
			attemptedAt,
		)
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseMarketInstrumentQuery(r *http.Request) (data.MarketInstrumentQuery, error) {
	values := r.URL.Query()
	query := data.MarketInstrumentQuery{
		Exchange: strings.TrimSpace(values.Get("exchange")),
		Query:    strings.ToUpper(strings.TrimSpace(values.Get("q"))),
		Limit:    20,
		Status:   "active",
	}
	if query.Exchange == "" {
		return data.MarketInstrumentQuery{}, fmt.Errorf("exchange is required")
	}
	if err := validateExchangeSymbol(query.Exchange, defaultSymbolForExchange(query.Exchange)); err != nil {
		return data.MarketInstrumentQuery{}, err
	}
	if rawLimit := values.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return data.MarketInstrumentQuery{}, fmt.Errorf("limit must be a positive integer")
		}
		if limit > maxMarketInstrumentQueryLimit {
			limit = maxMarketInstrumentQueryLimit
		}
		query.Limit = limit
	}
	if rawStatus := strings.ToLower(strings.TrimSpace(values.Get("status"))); rawStatus != "" {
		if rawStatus != "active" && rawStatus != "inactive" && rawStatus != "all" {
			return data.MarketInstrumentQuery{}, fmt.Errorf("status must be active, inactive or all")
		}
		query.Status = rawStatus
	}
	return query, nil
}

func parseMarketCandleGapScanQuery(r *http.Request) (data.MarketCandleGapScanQuery, error) {
	values := r.URL.Query()
	query := data.MarketCandleGapScanQuery{
		Exchange: strings.TrimSpace(values.Get("exchange")),
		Symbol:   strings.ToUpper(strings.TrimSpace(values.Get("symbol"))),
		Interval: strings.TrimSpace(values.Get("interval")),
		Limit:    data.DefaultMarketCandleGapScanLimit,
	}
	if query.Exchange == "" || query.Symbol == "" || query.Interval == "" {
		return data.MarketCandleGapScanQuery{}, fmt.Errorf("exchange, symbol and interval are required")
	}
	if err := validateExchangeSymbol(query.Exchange, query.Symbol); err != nil {
		return data.MarketCandleGapScanQuery{}, err
	}
	if _, err := data.IntervalDuration(query.Interval); err != nil {
		return data.MarketCandleGapScanQuery{}, err
	}
	if rawLimit := values.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return data.MarketCandleGapScanQuery{}, fmt.Errorf("limit must be a positive integer")
		}
		if limit > data.MaxMarketCandleGapScanLimit {
			return data.MarketCandleGapScanQuery{}, fmt.Errorf("limit must be less than or equal to %d", data.MaxMarketCandleGapScanLimit)
		}
		query.Limit = limit
	}
	return query, nil
}

func parseMarketCandleInvalidIssueScanQuery(r *http.Request) (data.MarketCandleInvalidIssueScanQuery, error) {
	values := r.URL.Query()
	query := data.MarketCandleInvalidIssueScanQuery{
		Exchange: strings.TrimSpace(values.Get("exchange")),
		Symbol:   strings.ToUpper(strings.TrimSpace(values.Get("symbol"))),
		Interval: strings.TrimSpace(values.Get("interval")),
		Limit:    data.DefaultMarketCandleInvalidIssueScanLimit,
	}
	if query.Exchange == "" || query.Symbol == "" || query.Interval == "" {
		return data.MarketCandleInvalidIssueScanQuery{}, fmt.Errorf("exchange, symbol and interval are required")
	}
	if err := validateExchangeSymbol(query.Exchange, query.Symbol); err != nil {
		return data.MarketCandleInvalidIssueScanQuery{}, err
	}
	if _, err := data.IntervalDuration(query.Interval); err != nil {
		return data.MarketCandleInvalidIssueScanQuery{}, err
	}
	if rawLimit := values.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return data.MarketCandleInvalidIssueScanQuery{}, fmt.Errorf("limit must be a positive integer")
		}
		if limit > data.MaxMarketCandleInvalidIssueScanLimit {
			return data.MarketCandleInvalidIssueScanQuery{}, fmt.Errorf("limit must be less than or equal to %d", data.MaxMarketCandleInvalidIssueScanLimit)
		}
		query.Limit = limit
	}
	return query, nil
}

func validateRepairMarketCandleGapRequest(request *data.RepairMarketCandleGapRequest) error {
	request.Exchange = strings.TrimSpace(request.Exchange)
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Interval = strings.TrimSpace(request.Interval)
	if request.Exchange == "" || request.Symbol == "" || request.Interval == "" {
		return fmt.Errorf("exchange, symbol and interval are required")
	}
	if err := validateExchangeSymbol(request.Exchange, request.Symbol); err != nil {
		return err
	}
	if _, err := data.IntervalDuration(request.Interval); err != nil {
		return err
	}
	if request.From.IsZero() || request.To.IsZero() || !request.From.Before(request.To) {
		return fmt.Errorf("from and to are required and from must be before to")
	}
	from := request.From.UTC()
	to := request.To.UTC()
	if err := data.ValidateDataSyncTaskWindow(request.Interval, &from, &to); err != nil {
		return err
	}
	return nil
}

func validateRepairMarketCandleGapsRequest(request *data.RepairMarketCandleGapsRequest) error {
	request.Exchange = strings.TrimSpace(request.Exchange)
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Interval = strings.TrimSpace(request.Interval)
	if request.Exchange == "" || request.Symbol == "" || request.Interval == "" {
		return fmt.Errorf("exchange, symbol and interval are required")
	}
	if len(request.Gaps) == 0 {
		return fmt.Errorf("gaps are required")
	}
	if len(request.Gaps) > data.MaxMarketCandleGapScanLimit {
		return fmt.Errorf("gaps must contain at most %d items", data.MaxMarketCandleGapScanLimit)
	}
	for index := range request.Gaps {
		gap := &request.Gaps[index]
		if gap.From.IsZero() || gap.To.IsZero() || !gap.From.Before(gap.To) {
			return fmt.Errorf("gap from and to are required and from must be before to")
		}
	}
	return validateRepairMarketCandleGapRequest(&data.RepairMarketCandleGapRequest{
		Exchange: request.Exchange,
		Symbol:   request.Symbol,
		Interval: request.Interval,
		From:     request.Gaps[0].From,
		To:       request.Gaps[0].To,
	})
}

func validateRepairMarketCandleInvalidIssuesRequest(request *data.RepairMarketCandleInvalidIssuesRequest) error {
	request.Exchange = strings.TrimSpace(request.Exchange)
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Interval = strings.TrimSpace(request.Interval)
	if request.Exchange == "" || request.Symbol == "" || request.Interval == "" {
		return fmt.Errorf("exchange, symbol and interval are required")
	}
	if err := validateExchangeSymbol(request.Exchange, request.Symbol); err != nil {
		return err
	}
	if err := data.ValidateDataSyncTaskWindow(request.Interval, nil, nil); err != nil {
		return err
	}
	if len(request.OpenTimes) == 0 {
		return fmt.Errorf("openTimes are required")
	}
	if len(request.OpenTimes) > data.MaxMarketCandleInvalidIssueScanLimit {
		return fmt.Errorf("openTimes must contain at most %d items", data.MaxMarketCandleInvalidIssueScanLimit)
	}
	for index := range request.OpenTimes {
		if request.OpenTimes[index].IsZero() {
			return fmt.Errorf("openTimes must not contain zero time")
		}
		request.OpenTimes[index] = request.OpenTimes[index].UTC()
	}
	return nil
}

func parseMarketInstrumentSyncQuery(r *http.Request) (data.MarketInstrumentQuery, error) {
	exchange := strings.TrimSpace(r.URL.Query().Get("exchange"))
	if exchange == "" {
		return data.MarketInstrumentQuery{}, fmt.Errorf("exchange is required")
	}
	if err := validateExchangeSymbol(exchange, defaultSymbolForExchange(exchange)); err != nil {
		return data.MarketInstrumentQuery{}, err
	}
	return data.MarketInstrumentQuery{Exchange: exchange}, nil
}

func defaultSymbolForExchange(exchange string) string {
	if exchange == "okx" {
		return "BTC-USDT"
	}
	return "BTCUSDT"
}
