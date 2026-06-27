package trading

import (
	"context"
	"errors"
	"time"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

var ErrLiveExecutionDisabled = errors.New("live execution is disabled until the live safety stage")

type OrderExecutor interface {
	Execute(ctx context.Context, request OrderRequest) (OrderExecution, error)
}

type OrderRequest struct {
	Task           data.TradingTask
	Intent         strategy.Intent
	IntentID       string
	IdempotencyKey string
	Now            time.Time
}

type OrderExecution struct {
	Order     data.Order
	Execution data.Execution
}

type PaperExecutor struct{}

func (PaperExecutor) Execute(_ context.Context, request OrderRequest) (OrderExecution, error) {
	orderID := core.StablePrefixedID("ord", "order:"+request.IdempotencyKey)
	executionID := core.StablePrefixedID("exe", "execution:"+request.IdempotencyKey)
	return OrderExecution{
		Order: data.Order{
			ID:             orderID,
			TaskID:         request.Task.ID,
			TaskType:       request.Task.Type,
			IntentID:       request.IntentID,
			IdempotencyKey: request.IdempotencyKey,
			Exchange:       request.Task.Exchange,
			AccountID:      request.Task.AccountID,
			Symbol:         request.Task.Symbol,
			Side:           request.Intent.Side,
			OrderType:      "market",
			Price:          request.Intent.Price,
			Quantity:       request.Intent.Quantity,
			Status:         "filled",
			ExchangeResponseSummary: map[string]any{
				"executor":   "paper",
				"source":     "local-simulation",
				"executedAt": request.Now,
			},
			CreatedAt: request.Now,
			UpdatedAt: request.Now,
		},
		Execution: data.Execution{
			ID:             executionID,
			TaskID:         request.Task.ID,
			TaskType:       request.Task.Type,
			OrderID:        orderID,
			IntentID:       request.IntentID,
			IdempotencyKey: request.IdempotencyKey,
			Exchange:       request.Task.Exchange,
			AccountID:      request.Task.AccountID,
			Symbol:         request.Task.Symbol,
			Side:           request.Intent.Side,
			Price:          request.Intent.Price,
			Quantity:       request.Intent.Quantity,
			Fee:            "0",
			Status:         "filled",
			ExecutedAt:     request.Now,
			CreatedAt:      request.Now,
		},
	}, nil
}

type LiveExecutor struct{}

func (LiveExecutor) Execute(context.Context, OrderRequest) (OrderExecution, error) {
	return OrderExecution{}, ErrLiveExecutionDisabled
}
