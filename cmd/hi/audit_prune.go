package main

import (
	"context"
	"log/slog"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/store/postgres"
)

func runAuditPrune(ctx context.Context, args []string) error {
	config, err := loadAuditPruneCommandConfig(args)
	if err != nil {
		return err
	}

	store, err := postgres.OpenWithOptions(ctx, config.DatabaseURL, config.DatabasePool)
	if err != nil {
		return err
	}
	defer store.Close()

	result, err := store.PruneAuditEvents(ctx, data.AuditEventRetentionRequest{
		Before: config.Cutoff,
		DryRun: config.DryRun,
	})
	if err != nil {
		return err
	}
	slog.Info("audit prune completed", safeConfigSummary(
		"dry_run", result.DryRun,
		"retention_days", config.RetentionDays,
		"cutoff", result.Cutoff,
		"pruned_count", result.PrunedCount,
		"hashed_pruned_count", result.HashedPrunedCount,
		"legacy_pruned_count", result.LegacyPrunedCount,
		"anchor_id", result.AnchorID,
		"anchor_event_id", result.AnchorEventID,
		"retained_event_id", result.RetainedEventID,
		"message", result.Message,
	)...)
	return nil
}
