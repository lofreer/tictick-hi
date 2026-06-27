package api

import (
	"net/http"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

func (server *Server) handleBacktests(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 2 {
		server.handleBacktestCollection(w, r)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet {
		server.getBacktest(w, r, parts[2])
		return
	}
	if len(parts) == 4 && parts[3] == "orders" && r.Method == http.MethodGet {
		server.listBacktestOrders(w, r, parts[2])
		return
	}
	writeError(w, http.StatusNotFound, "backtest route not found")
}

func (server *Server) handleBacktestCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasks, err := server.repository.ListBacktestTasks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		var request data.CreateBacktestTask
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		normalizeCreateBacktest(&request)
		definition, err := server.strategyRepository.GetStrategy(r.Context(), request.StrategyID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := validateCreateBacktest(request, definition); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		normalizedParams, err := strategy.NormalizeParams(definition, request.StrategyParams)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		request.StrategyParams = normalizedParams
		task, err := server.repository.CreateBacktestTask(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) getBacktest(w http.ResponseWriter, r *http.Request, id string) {
	task, err := server.repository.GetBacktestTask(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) listBacktestOrders(w http.ResponseWriter, r *http.Request, id string) {
	if _, err := server.repository.GetBacktestTask(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	orders, err := server.repository.ListBacktestOrders(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orders)
}
