package main

import (
	"context"

	"github.com/lofreer/tictick-hi/internal/store/postgres"
)

func loadWorkerProbeRuntimeConfig(command string) (string, workerReadinessBacklogConfig, error) {
	healthAddr, err := loadWorkerHealthProbeAddr(command)
	if err != nil {
		return "", workerReadinessBacklogConfig{}, err
	}
	backlogReadiness, err := loadWorkerReadinessBacklogConfig(command)
	if err != nil {
		return "", workerReadinessBacklogConfig{}, err
	}
	return healthAddr, backlogReadiness, nil
}

func workerReadinessChecks(
	store *postgres.Store,
	command string,
	backlogReadiness workerReadinessBacklogConfig,
) []workerReadinessCheck {
	checks := []workerReadinessCheck{
		{
			Name:  "postgres",
			Check: store.Ping,
		},
		{
			Name: "queue",
			Check: func(ctx context.Context) error {
				return store.CheckWorkerQueue(ctx, command)
			},
		},
	}
	if backlogReadiness.enabled() {
		checks = append(checks, workerReadinessCheck{
			Name: "queue_backlog",
			Check: func(ctx context.Context) error {
				return store.CheckWorkerQueueBacklog(ctx, command, backlogReadiness.limits())
			},
		})
	}
	return checks
}
