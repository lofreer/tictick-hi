package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

const (
	sessionCookieName = "tictick_hi_session"
	csrfCookieName    = "tictick_hi_csrf"
	csrfHeaderName    = "X-CSRF-Token"
)

type Config struct {
	StaticRoot         string
	SessionTTL         time.Duration
	CookieSecure       bool
	LoginFailureLimit  int
	LoginFailureWindow time.Duration
	LoginLockout       time.Duration
	InstrumentClients  map[string]exchange.InstrumentClient
}

type Server struct {
	repository         data.Repository
	strategyRepository strategy.Repository
	instrumentClients  map[string]exchange.InstrumentClient
	staticRoot         string
	sessionTTL         time.Duration
	cookieSecure       bool
	loginLimiter       *loginLimiter
}

func NewServer(repository data.Repository, staticRoot string) http.Handler {
	return NewServerWithConfig(repository, Config{StaticRoot: staticRoot})
}

func NewServerWithConfig(repository data.Repository, config Config) http.Handler {
	if config.SessionTTL <= 0 {
		config.SessionTTL = 12 * time.Hour
	}
	if config.LoginFailureLimit <= 0 {
		config.LoginFailureLimit = 5
	}
	if config.LoginFailureWindow <= 0 {
		config.LoginFailureWindow = 5 * time.Minute
	}
	if config.LoginLockout <= 0 {
		config.LoginLockout = 5 * time.Minute
	}
	return &Server{
		repository:         repository,
		strategyRepository: strategy.BuiltinRegistry(),
		instrumentClients:  cloneInstrumentClients(config.InstrumentClients),
		staticRoot:         config.StaticRoot,
		sessionTTL:         config.SessionTTL,
		cookieSecure:       config.CookieSecure,
		loginLimiter:       newLoginLimiter(config.LoginFailureLimit, config.LoginFailureWindow, config.LoginLockout),
	}
}

func cloneInstrumentClients(clients map[string]exchange.InstrumentClient) map[string]exchange.InstrumentClient {
	if len(clients) == 0 {
		return nil
	}
	cloned := make(map[string]exchange.InstrumentClient, len(clients))
	for key, client := range clients {
		cloned[key] = client
	}
	return cloned
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request, err := withRequestID(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	r = request

	switch {
	case r.URL.Path == "/readyz":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case strings.HasPrefix(r.URL.Path, "/api/auth"):
		server.handleAuth(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/"):
		if _, ok := server.authenticateRequest(w, r); !ok {
			return
		}
		if !server.validateCSRF(w, r) {
			return
		}
		server.serveAPI(w, r)
	default:
		server.serveFrontend(w, r)
	}
}

func (server *Server) serveAPI(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/api/data/tasks"):
		server.handleDataTasks(w, r)
	case r.URL.Path == "/api/candles":
		server.handleCandles(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/market"):
		server.handleMarket(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/overview"):
		server.handleOverview(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/strategies"):
		server.handleStrategies(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/backtests"):
		server.handleBacktests(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/trading/tasks"):
		server.handleTradingTasks(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/system"):
		server.handleSystem(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/"):
		writeError(w, http.StatusNotFound, "api route not found")
	default:
		server.serveFrontend(w, r)
	}
}
