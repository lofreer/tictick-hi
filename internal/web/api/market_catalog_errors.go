package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	marketInstrumentNotActiveMessage = "market instrument is not active in catalog"
	marketInstrumentInactiveMessage  = "market instrument is inactive in catalog"
	marketInstrumentMissingMessage   = "market instrument is missing from catalog"
)

func isNotFoundError(err error) bool {
	return errors.Is(err, data.ErrNotFound)
}

func (server *Server) marketInstrumentCatalogMessage(
	r *http.Request,
	exchange string,
	symbol string,
) string {
	requestedSymbol := strings.ToUpper(strings.TrimSpace(symbol))
	instruments, err := server.repository.ListMarketInstruments(r.Context(), data.MarketInstrumentQuery{
		Exchange: exchange,
		Query:    requestedSymbol,
		Status:   "all",
		Limit:    maxMarketInstrumentQueryLimit,
	})
	if err != nil {
		return marketInstrumentNotActiveMessage
	}
	for _, instrument := range instruments {
		if instrument.Exchange != exchange || !strings.EqualFold(strings.TrimSpace(instrument.Symbol), requestedSymbol) {
			continue
		}
		if instrument.Status == "active" {
			return marketInstrumentNotActiveMessage
		}
		return marketInstrumentInactiveMessage
	}
	return marketInstrumentMissingMessage
}
