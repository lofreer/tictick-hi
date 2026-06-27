package notification

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

type Provider interface {
	Deliver(ctx context.Context, delivery data.NotificationDelivery) error
}

type ProviderRegistry struct {
	providers map[string]Provider
}

func DemoProviders() ProviderRegistry {
	provider := DemoProvider{}
	return ProviderRegistry{providers: map[string]Provider{
		"local":        provider,
		"webhook-demo": provider,
	}}
}

func (registry ProviderRegistry) Provider(name string) (Provider, error) {
	provider, ok := registry.providers[name]
	if !ok {
		return nil, fmt.Errorf("notification provider %q is not registered", name)
	}
	return provider, nil
}

type DemoProvider struct{}

func (DemoProvider) Deliver(_ context.Context, delivery data.NotificationDelivery) error {
	if delivery.Target == "" {
		return errors.New("notification target is required")
	}
	if strings.Contains(strings.ToLower(delivery.Target), "fail") {
		return fmt.Errorf("demo provider rejected target %q", delivery.Target)
	}
	return nil
}
