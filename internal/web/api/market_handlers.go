package api

import (
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
	case "/api/market/instruments":
		server.handleMarketInstruments(w, r)
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
	instruments, err := client.FetchInstruments(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, apiErrorRequestFailed, "sync market instruments: "+err.Error())
		return
	}
	result, err := server.repository.ReplaceMarketInstruments(r.Context(), query.Exchange, instruments, time.Now().UTC())
	if err != nil {
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
