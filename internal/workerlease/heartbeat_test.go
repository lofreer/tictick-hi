package workerlease

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunWithHeartbeatRunsInitialHeartbeatBeforeWork(t *testing.T) {
	heartbeats := 0
	err := RunWithHeartbeat(
		context.Background(),
		time.Hour,
		func(context.Context) error {
			heartbeats++
			return nil
		},
		func(context.Context) error {
			if heartbeats != 1 {
				t.Fatalf("heartbeats before work = %d, want 1", heartbeats)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunWithHeartbeatRefreshesWhileWorkRuns(t *testing.T) {
	heartbeatSignals := make(chan struct{}, 4)
	err := RunWithHeartbeat(
		context.Background(),
		time.Millisecond,
		func(context.Context) error {
			select {
			case heartbeatSignals <- struct{}{}:
			default:
			}
			return nil
		},
		func(ctx context.Context) error {
			for index := 0; index < 2; index++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-heartbeatSignals:
				}
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunWithHeartbeatCancelsWorkOnHeartbeatError(t *testing.T) {
	heartbeatErr := errors.New("lease lost")
	heartbeats := 0
	err := RunWithHeartbeat(
		context.Background(),
		time.Millisecond,
		func(context.Context) error {
			heartbeats++
			if heartbeats > 1 {
				return heartbeatErr
			}
			return nil
		},
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	)
	if !errors.Is(err, heartbeatErr) {
		t.Fatalf("error = %v, want %v", err, heartbeatErr)
	}
}
