package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

const maxMarketInstrumentQueryLimit = 50

func (server *Server) handleMarket(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/market/instruments" {
		writeError(w, http.StatusNotFound, "market route not found")
		return
	}
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

func defaultSymbolForExchange(exchange string) string {
	if exchange == "okx" {
		return "BTC-USDT"
	}
	return "BTCUSDT"
}
