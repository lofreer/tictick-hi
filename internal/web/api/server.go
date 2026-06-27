package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

const sessionCookieName = "tictick_hi_session"

type Config struct {
	StaticRoot   string
	SessionTTL   time.Duration
	CookieSecure bool
}

type Server struct {
	repository         data.Repository
	strategyRepository strategy.Repository
	staticRoot         string
	sessionTTL         time.Duration
	cookieSecure       bool
}

func NewServer(repository data.Repository, staticRoot string) http.Handler {
	return NewServerWithConfig(repository, Config{StaticRoot: staticRoot})
}

func NewServerWithConfig(repository data.Repository, config Config) http.Handler {
	if config.SessionTTL <= 0 {
		config.SessionTTL = 12 * time.Hour
	}
	return &Server{
		repository:         repository,
		strategyRepository: strategy.BuiltinRegistry(),
		staticRoot:         config.StaticRoot,
		sessionTTL:         config.SessionTTL,
		cookieSecure:       config.CookieSecure,
	}
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/readyz":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case strings.HasPrefix(r.URL.Path, "/api/auth"):
		server.handleAuth(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/"):
		if _, ok := server.authenticateRequest(w, r); !ok {
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
