package exchange

import (
	"context"
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type CandleRequest struct {
	Exchange string
	Symbol   string
	Interval string
	From     time.Time
	To       time.Time
	Limit    int
}

type MarketDataClient interface {
	FetchCandles(ctx context.Context, request CandleRequest) ([]data.Candle, error)
}

type InstrumentClient interface {
	FetchInstruments(ctx context.Context) ([]data.MarketInstrument, error)
}

type Registry struct {
	clients map[string]MarketDataClient
}

func NewRegistry(clients map[string]MarketDataClient) Registry {
	return Registry{clients: clients}
}

func (registry Registry) Client(exchange string) (MarketDataClient, error) {
	client, ok := registry.clients[exchange]
	if !ok {
		return nil, fmt.Errorf("exchange %q is not registered", exchange)
	}
	return client, nil
}
