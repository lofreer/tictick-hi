package api

import (
	"net/http"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

func (server *Server) handleTradingTasks(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 3 {
		server.handleTradingTaskCollection(w, r)
		return
	}
	if len(parts) == 4 && r.Method == http.MethodGet {
		server.getTradingTask(w, r, parts[3])
		return
	}
	if len(parts) == 5 && r.Method == http.MethodPost {
		server.handleTradingTaskCommand(w, r, parts[3], parts[4])
		return
	}
	if len(parts) == 5 && r.Method == http.MethodGet {
		server.handleTradingTaskDetailCollection(w, r, parts[3], parts[4])
		return
	}
	writeError(w, http.StatusNotFound, "trading task route not found")
}

func (server *Server) handleTradingTaskCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasks, err := server.repository.ListTradingTasks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		var request data.CreateTradingTask
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		normalizeCreateTradingTask(&request)
		definition, err := server.strategyRepository.GetStrategy(r.Context(), request.StrategyID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := validateCreateTradingTask(request, definition); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		normalizedParams, err := strategy.NormalizeParams(definition, request.StrategyParams)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		request.StrategyParams = normalizedParams
		task, err := server.repository.CreateTradingTask(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) getTradingTask(w http.ResponseWriter, r *http.Request, id string) {
	task, err := server.repository.GetTradingTask(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) handleTradingTaskCommand(w http.ResponseWriter, r *http.Request, id string, action string) {
	status := data.TaskStatus("")
	switch action {
	case "start":
		status = data.TaskStatusRunning
	case "pause", "stop":
		status = data.TaskStatusPaused
	default:
		writeError(w, http.StatusNotFound, "trading task command not found")
		return
	}
	task, err := server.repository.SetTradingTaskStatus(r.Context(), id, status)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) handleTradingTaskDetailCollection(
	w http.ResponseWriter,
	r *http.Request,
	id string,
	collection string,
) {
	if _, err := server.repository.GetTradingTask(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	switch collection {
	case "intents":
		intents, err := server.repository.ListTradingIntents(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, intents)
	case "orders":
		orders, err := server.repository.ListTradingOrders(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, orders)
	case "executions":
		executions, err := server.repository.ListTradingExecutions(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, executions)
	case "positions":
		positions, err := server.repository.ListTradingPositions(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, positions)
	case "notifications":
		notifications, err := server.repository.ListTradingNotifications(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, notifications)
	default:
		writeError(w, http.StatusNotFound, "trading task collection not found")
	}
}
