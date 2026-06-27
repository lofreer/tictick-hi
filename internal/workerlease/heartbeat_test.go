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

func TestIsShutdownRequiresParentContextCancellation(t *testing.T) {
	if IsShutdown(context.Background(), context.Canceled) {
		t.Fatal("context.Canceled from work should not be treated as shutdown unless parent context is canceled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !IsShutdown(ctx, context.Canceled) {
		t.Fatal("expected canceled parent context to be treated as shutdown")
	}
	if !IsShutdown(ctx, errors.New("transport closed after signal")) {
		t.Fatal("expected any work error after parent cancellation to be treated as shutdown")
	}
}

func TestReleaseContextIgnoresParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	releaseCtx, releaseCancel := ReleaseContext(ctx)
	defer releaseCancel()

	if err := releaseCtx.Err(); err != nil {
		t.Fatalf("release context should not inherit parent cancellation: %v", err)
	}
}
