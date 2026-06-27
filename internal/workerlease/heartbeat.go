package workerlease

import (
	"context"
	"errors"
	"time"
)

type HeartbeatFunc func(context.Context) error
type WorkFunc func(context.Context) error

const releaseTimeout = 5 * time.Second

func IsShutdown(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() == nil {
		return false
	}
	return true
}

func ReleaseContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
}

func RunWithHeartbeat(
	ctx context.Context,
	interval time.Duration,
	heartbeat HeartbeatFunc,
	work WorkFunc,
) error {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	runCtx, cancel := context.WithCancel(ctx)
	heartbeatErrors := make(chan error, 1)
	done := make(chan struct{})

	if err := heartbeat(runCtx); err != nil {
		cancel()
		return err
	}

	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if err := heartbeat(runCtx); err != nil {
					select {
					case heartbeatErrors <- err:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()

	workErr := work(runCtx)
	cancel()
	<-done

	select {
	case heartbeatErr := <-heartbeatErrors:
		if workErr == nil || errors.Is(workErr, context.Canceled) {
			return heartbeatErr
		}
	default:
	}
	return workErr
}
