package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (server *Server) quarantineMarketCandleInvalidIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var request data.QuarantineMarketCandleInvalidIssuesRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateQuarantineMarketCandleInvalidIssuesRequest(&request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := server.repository.QuarantineMarketCandleInvalidIssues(r.Context(), request)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func validateQuarantineMarketCandleInvalidIssuesRequest(
	request *data.QuarantineMarketCandleInvalidIssuesRequest,
) error {
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
