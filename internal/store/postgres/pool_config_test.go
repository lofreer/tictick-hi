package postgres

import (
	"strings"
	"testing"
	"time"
)

func TestNewPoolConfigAppliesOptions(t *testing.T) {
	config, err := newPoolConfig("postgres://user:pass@localhost:5432/tictick_hi?sslmode=disable", PoolOptions{
		MaxConns:        7,
		MinConns:        2,
		MaxConnLifetime: 45 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("new pool config: %v", err)
	}
	if config.MaxConns != 7 ||
		config.MinConns != 2 ||
		config.MaxConnLifetime != 45*time.Minute ||
		config.MaxConnIdleTime != 5*time.Minute {
		t.Fatalf("unexpected pool config: %#v", config)
	}
}

func TestNewPoolConfigRejectsInvalidOptions(t *testing.T) {
	_, err := newPoolConfig("postgres://user:pass@localhost:5432/tictick_hi?sslmode=disable", PoolOptions{
		MaxConns:        2,
		MinConns:        3,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: time.Minute,
	})
	if err == nil || !strings.Contains(err.Error(), "min conns") {
		t.Fatalf("expected min conns error, got %v", err)
	}
}
