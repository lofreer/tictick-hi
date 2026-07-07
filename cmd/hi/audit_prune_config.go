package main

import (
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/store/postgres"
)

const maxAuditRetentionDays = 36500

type auditPruneCommandConfig struct {
	DatabaseURL   string
	DatabasePool  postgres.PoolOptions
	DryRun        bool
	RetentionDays int
	Cutoff        time.Time
}

func loadAuditPruneCommandConfig(args []string) (auditPruneCommandConfig, error) {
	flags := newCommandFlagSet("audit-prune")
	retentionDaysFlag := flags.Int("retention-days", 0, "audit event retention window in days")
	execute := flags.Bool("execute", false, "delete eligible audit events")
	if err := flags.Parse(args); err != nil {
		return auditPruneCommandConfig{}, err
	}

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return auditPruneCommandConfig{}, err
	}
	databasePool, err := loadDatabasePoolOptions()
	if err != nil {
		return auditPruneCommandConfig{}, err
	}
	retentionDays, err := resolveAuditRetentionDays(*retentionDaysFlag)
	if err != nil {
		return auditPruneCommandConfig{}, err
	}
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	return auditPruneCommandConfig{
		DatabaseURL:   databaseURL,
		DatabasePool:  databasePool,
		DryRun:        !*execute,
		RetentionDays: retentionDays,
		Cutoff:        cutoff,
	}, nil
}

func resolveAuditRetentionDays(flagValue int) (int, error) {
	if flagValue < 0 {
		return 0, fmt.Errorf("--retention-days must be greater than or equal to 1")
	}
	if flagValue > 0 {
		return validateAuditRetentionDays("--retention-days", flagValue)
	}
	days, err := intEnvStrict("AUDIT_RETENTION_DAYS", 0, 1)
	if err != nil {
		return 0, err
	}
	if days == 0 {
		return 0, fmt.Errorf("AUDIT_RETENTION_DAYS is required when --retention-days is omitted")
	}
	return validateAuditRetentionDays("AUDIT_RETENTION_DAYS", days)
}

func validateAuditRetentionDays(name string, days int) (int, error) {
	if days > maxAuditRetentionDays {
		return 0, fmt.Errorf("%s must be less than or equal to %d", name, maxAuditRetentionDays)
	}
	return days, nil
}
