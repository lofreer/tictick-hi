package api

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) ListNativeCandles(
	_ context.Context,
	query data.CandleQuery,
) ([]data.Candle, error) {
	matches := make([]data.Candle, 0)
	for _, candle := range repository.candles {
		if candle.Exchange != query.Exchange || candle.Symbol != query.Symbol || candle.Interval != query.Interval {
			continue
		}
		if query.From != nil && candle.OpenTime.Before(*query.From) {
			continue
		}
		if query.To != nil && candle.OpenTime.After(*query.To) {
			continue
		}
		matches = append(matches, candle)
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})
	if limit := data.NormalizeCandleLimit(query.Limit); len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func (repository *fakeRepository) ScanMarketCandleGaps(
	_ context.Context,
	query data.MarketCandleGapScanQuery,
) (data.MarketCandleGapScan, error) {
	matches := make([]data.Candle, 0)
	for _, candle := range repository.candles {
		if candle.Exchange == query.Exchange && candle.Symbol == query.Symbol && candle.Interval == query.Interval {
			matches = append(matches, candle)
		}
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})

	result := data.MarketCandleGapScan{
		Exchange: query.Exchange,
		Symbol:   query.Symbol,
		Interval: query.Interval,
		Gaps:     []data.CandleGap{},
		Window: data.CandleWindow{
			Count: len(matches),
		},
	}
	if len(matches) > 0 {
		from := matches[0].OpenTime.UTC()
		to := matches[len(matches)-1].OpenTime.UTC()
		result.Window.From = &from
		result.Window.To = &to
	}

	gaps, err := data.DetectCandleGaps(matches, query.Interval)
	if err != nil {
		return data.MarketCandleGapScan{}, err
	}
	limit := data.NormalizeMarketCandleGapScanLimit(query.Limit)
	result.TotalCount = len(gaps)
	result.Limited = len(gaps) > limit
	if len(gaps) > limit {
		gaps = gaps[:limit]
	}
	result.Gaps = gaps
	result.ReturnedCount = len(gaps)
	return result, nil
}

func (repository *fakeRepository) RepairMarketCandleGap(
	_ context.Context,
	request data.RepairMarketCandleGapRequest,
) (data.DataSyncGapRepairResult, error) {
	gap, ok, err := repository.fakeMarketCandleGap(request)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if !ok {
		return data.DataSyncGapRepairResult{}, data.ErrNotFound
	}

	result := data.DataSyncGapRepairResult{
		CreatedTasks: []data.DataSyncTask{},
		TotalCount:   1,
		RepairLimit:  1,
	}
	source := data.DataSyncTask{
		Exchange: request.Exchange,
		Symbol:   request.Symbol,
		Interval: request.Interval,
	}
	if repository.fakeRepairTaskExists(source, gap) {
		result.SkippedExisting = 1
		return result, nil
	}

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repairTask := data.DataSyncTask{
		ID:              "dst_market_repair_" + strconv.Itoa(len(repository.tasks)+1),
		Exchange:        request.Exchange,
		Symbol:          request.Symbol,
		Interval:        request.Interval,
		StartTime:       &gap.From,
		EndTime:         &gap.To,
		SyncEnabled:     true,
		RealtimeEnabled: false,
		Status:          data.TaskStatusPending,
		DataHealth:      data.DataSyncHealthSyncing,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	repository.tasks = append(repository.tasks, repairTask)
	result.CreatedTasks = append(result.CreatedTasks, repairTask)
	return result, nil
}

func (repository *fakeRepository) fakeMarketCandleGap(
	request data.RepairMarketCandleGapRequest,
) (data.CandleGap, bool, error) {
	matches := make([]data.Candle, 0)
	for _, candle := range repository.candles {
		if candle.Exchange == request.Exchange && candle.Symbol == request.Symbol && candle.Interval == request.Interval {
			matches = append(matches, candle)
		}
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})
	gaps, err := data.DetectCandleGaps(matches, request.Interval)
	if err != nil {
		return data.CandleGap{}, false, err
	}
	for _, gap := range gaps {
		if gap.From.Equal(request.From) && gap.To.Equal(request.To) {
			return gap, true, nil
		}
	}
	return data.CandleGap{}, false, nil
}
