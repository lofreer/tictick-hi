package strategy

import (
	"context"

	"github.com/lofreer/tictick-hi/internal/data"
)

type Registry struct {
	strategies []Definition
}

type Definition struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Version            string      `json:"version"`
	Description        string      `json:"description"`
	SupportedIntervals []string    `json:"supportedIntervals"`
	SupportedIntents   []string    `json:"supportedIntents"`
	Params             []ParamSpec `json:"params"`
}

type ParamSpec struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Default     any      `json:"default,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Step        *float64 `json:"step,omitempty"`
	Options     []Option `json:"options,omitempty"`
	Description string   `json:"description,omitempty"`
}

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Repository interface {
	ListStrategies(ctx context.Context) ([]Definition, error)
	GetStrategy(ctx context.Context, id string) (Definition, error)
}

func BuiltinRegistry() Registry {
	return Registry{strategies: []Definition{
		{
			ID:                 "ema-cross",
			Name:               "EMA Cross",
			Version:            "v1",
			Description:        "Generate order intents when fast and slow EMA lines cross.",
			SupportedIntervals: []string{"1m", "5m", "15m", "1h", "4h", "1d"},
			SupportedIntents:   []string{"order", "notification"},
			Params: []ParamSpec{
				intParam("fastPeriod", "Fast EMA Period", 12, 2, 200, 1),
				intParam("slowPeriod", "Slow EMA Period", 26, 3, 400, 1),
				numberParam("orderSize", "Order Size", 0.01, 0.0001, 100, 0.0001),
				selectParam("signalMode", "Signal Mode", "order", []Option{
					{Label: "Order intent", Value: "order"},
					{Label: "Notification only", Value: "notification"},
				}),
			},
		},
		{
			ID:                 "breakout-range",
			Name:               "Range Breakout",
			Version:            "v1",
			Description:        "Emit intents when price breaks out of a rolling high/low range.",
			SupportedIntervals: []string{"5m", "15m", "1h", "4h", "1d"},
			SupportedIntents:   []string{"order", "notification"},
			Params: []ParamSpec{
				intParam("lookback", "Lookback Candles", 20, 5, 300, 1),
				numberParam("breakoutBufferPct", "Breakout Buffer %", 0.2, 0, 10, 0.1),
				numberParam("orderSize", "Order Size", 0.01, 0.0001, 100, 0.0001),
				selectParam("side", "Side", "both", []Option{
					{Label: "Both", Value: "both"},
					{Label: "Long only", Value: "long"},
					{Label: "Short only", Value: "short"},
				}),
			},
		},
	}}
}

func (registry Registry) ListStrategies(context.Context) ([]Definition, error) {
	return append([]Definition(nil), registry.strategies...), nil
}

func (registry Registry) GetStrategy(_ context.Context, id string) (Definition, error) {
	for _, definition := range registry.strategies {
		if definition.ID == id {
			return definition, nil
		}
	}
	return Definition{}, data.ErrNotFound
}

func intParam(key string, label string, defaultValue int, min float64, max float64, step float64) ParamSpec {
	return numberParam(key, label, defaultValue, min, max, step)
}

func numberParam(
	key string,
	label string,
	defaultValue any,
	min float64,
	max float64,
	step float64,
) ParamSpec {
	return ParamSpec{
		Key:      key,
		Label:    label,
		Type:     "number",
		Required: true,
		Default:  defaultValue,
		Min:      &min,
		Max:      &max,
		Step:     &step,
	}
}

func selectParam(key string, label string, defaultValue string, options []Option) ParamSpec {
	return ParamSpec{
		Key:      key,
		Label:    label,
		Type:     "select",
		Required: true,
		Default:  defaultValue,
		Options:  options,
	}
}
