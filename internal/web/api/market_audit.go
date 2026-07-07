package api

import (
	"net/http"
	"strconv"

	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	marketInstrumentSyncAuditAction       = "market_instrument.sync"
	marketInstrumentSyncAuditResourceType = "market_instrument_catalog"
)

func (server *Server) recordMarketInstrumentSyncAudit(
	r *http.Request,
	actor data.Operator,
	exchange string,
	outcome string,
	metadata map[string]string,
) error {
	return server.recordAuditEvent(
		r,
		actor,
		marketInstrumentSyncAuditAction,
		marketInstrumentSyncAuditResourceType,
		exchange,
		outcome,
		metadata,
	)
}

func marketInstrumentSyncFailureMetadata(exchange string, reason string) map[string]string {
	return map[string]string{
		"exchange": exchange,
		"reason":   reason,
	}
}

func marketInstrumentSyncStoreFailureMetadata(exchange string, err error) map[string]string {
	return storeFailureMetadata(err, map[string]string{
		"exchange": exchange,
	})
}

func marketInstrumentSyncResultAuditMetadata(result data.MarketInstrumentSyncResult) map[string]string {
	return map[string]string{
		"exchange":                  result.Exchange,
		"activeCount":               strconv.Itoa(result.ActiveCount),
		"inactiveCount":             strconv.Itoa(result.InactiveCount),
		"pausedDataSyncTaskCount":   strconv.Itoa(result.PausedDataSyncTaskCount),
		"restoredDataSyncTaskCount": strconv.Itoa(result.RestoredDataSyncTaskCount),
	}
}
