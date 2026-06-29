package postgres

import (
	"strings"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func assertIntegrationNoRequestURLLeak(t *testing.T, value string) {
	t.Helper()
	for _, forbidden := range []string{`Get "`, "https://", "/api/v3/klines", "symbol=BTCUSDT", "endTime=", "startTime="} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("stored error leaks %q: %s", forbidden, value)
		}
	}
}

func findIntegrationServiceHealth(health data.SystemHealth, name string) data.ServiceHealth {
	for _, service := range health.Services {
		if service.Name == name {
			return service
		}
	}
	return data.ServiceHealth{}
}
