package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/store/postgres"
)

func bootstrapOperator(ctx context.Context, store *postgres.Store) error {
	username := strings.TrimSpace(os.Getenv("BOOTSTRAP_OPERATOR_USERNAME"))
	password := os.Getenv("BOOTSTRAP_OPERATOR_PASSWORD")
	if username == "" && password == "" {
		return nil
	}
	if username == "" || password == "" {
		return fmt.Errorf("BOOTSTRAP_OPERATOR_USERNAME and BOOTSTRAP_OPERATOR_PASSWORD must be set together")
	}
	if err := data.ValidateOperatorPasswordForUsername(username, password); err != nil {
		return fmt.Errorf("BOOTSTRAP_OPERATOR_PASSWORD does not satisfy operator password policy: %w", err)
	}

	operator, created, err := store.EnsureOperator(ctx, data.CreateOperator{
		Username: username,
		Password: password,
		Role:     data.OperatorRoleAdmin,
		Enabled:  true,
	})
	if err != nil {
		return err
	}
	if created {
		slog.Info("bootstrapped operator", "username", operator.Username)
	}
	return nil
}
