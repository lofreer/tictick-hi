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
	if query.From == nil && query.To == nil {
		return repository.latestNativeCandles(query), nil
	}
	return repository.nativeCandles(query), nil
}

func (repository *fakeRepository) ListLatestNativeCandles(
	_ context.Context,
	query data.CandleQuery,
) ([]data.Candle, error) {
	return repository.latestNativeCandles(query), nil
}

func (repository *fakeRepository) nativeCandles(query data.CandleQuery) []data.Candle {
	matches := repository.matchingNativeCandles(query)
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})
	if limit := data.NormalizeCandleLimit(query.Limit); len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func (repository *fakeRepository) latestNativeCandles(query data.CandleQuery) []data.Candle {
	matches := repository.matchingNativeCandles(query)
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.After(matches[right].OpenTime)
	})
	if limit := data.NormalizeCandleLimit(query.Limit); len(matches) > limit {
		matches = matches[:limit]
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})
	return matches
}

func (repository *fakeRepository) matchingNativeCandles(query data.CandleQuery) []data.Candle {
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
	return matches
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

func (repository *fakeRepository) ScanMarketCandleInvalidIssues(
	_ context.Context,
	query data.MarketCandleInvalidIssueScanQuery,
) (data.MarketCandleInvalidIssueScan, error) {
	matches := make([]data.Candle, 0)
	for _, candle := range repository.candles {
		if candle.Exchange == query.Exchange && candle.Symbol == query.Symbol && candle.Interval == query.Interval {
			matches = append(matches, candle)
		}
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})

	result := data.MarketCandleInvalidIssueScan{
		Exchange: query.Exchange,
		Symbol:   query.Symbol,
		Interval: query.Interval,
		Issues:   []data.CandleIssue{},
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

	issues := make([]data.CandleIssue, 0)
	for _, candle := range matches {
		issue := data.DetectCandleIssue(candle)
		if issue == nil {
			continue
		}
		issues = append(issues, *issue)
	}
	limit := data.NormalizeMarketCandleInvalidIssueScanLimit(query.Limit)
	result.TotalCount = len(issues)
	result.Limited = len(issues) > limit
	if len(issues) > limit {
		issues = issues[:limit]
	}
	result.Issues = issues
	result.ReturnedCount = len(issues)
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
		RequestID:       request.RequestID,
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

func (repository *fakeRepository) RepairMarketCandleGaps(
	_ context.Context,
	request data.RepairMarketCandleGapsRequest,
) (data.DataSyncGapRepairResult, error) {
	result := data.DataSyncGapRepairResult{
		CreatedTasks: []data.DataSyncTask{},
		TotalCount:   len(request.Gaps),
		RepairLimit:  data.MaxMarketCandleGapScanLimit,
	}
	gaps := make([]data.CandleGap, 0, len(request.Gaps))
	for _, gapRequest := range request.Gaps {
		gap, ok, err := repository.fakeMarketCandleGap(data.RepairMarketCandleGapRequest{
			Exchange:  request.Exchange,
			Symbol:    request.Symbol,
			Interval:  request.Interval,
			From:      gapRequest.From,
			To:        gapRequest.To,
			RequestID: request.RequestID,
		})
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		if !ok {
			return data.DataSyncGapRepairResult{}, data.ErrNotFound
		}
		gaps = append(gaps, gap)
	}

	source := data.DataSyncTask{Exchange: request.Exchange, Symbol: request.Symbol, Interval: request.Interval}
	for _, gap := range gaps {
		if repository.fakeRepairTaskExists(source, gap) {
			result.SkippedExisting += 1
			continue
		}
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		startTime := gap.From
		endTime := gap.To
		repairTask := data.DataSyncTask{
			ID:              "dst_market_repair_" + strconv.Itoa(len(repository.tasks)+1),
			Exchange:        request.Exchange,
			Symbol:          request.Symbol,
			Interval:        request.Interval,
			StartTime:       &startTime,
			EndTime:         &endTime,
			RequestID:       request.RequestID,
			SyncEnabled:     true,
			RealtimeEnabled: false,
			Status:          data.TaskStatusPending,
			DataHealth:      data.DataSyncHealthSyncing,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		repository.tasks = append(repository.tasks, repairTask)
		result.CreatedTasks = append(result.CreatedTasks, repairTask)
	}
	return result, nil
}

func (repository *fakeRepository) RepairMarketCandleInvalidIssues(
	_ context.Context,
	request data.RepairMarketCandleInvalidIssuesRequest,
) (data.DataSyncGapRepairResult, error) {
	duration, err := data.IntervalDuration(request.Interval)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	result := data.DataSyncGapRepairResult{
		CreatedTasks: []data.DataSyncTask{},
		TotalCount:   len(request.OpenTimes),
		RepairLimit:  data.MaxMarketCandleInvalidIssueScanLimit,
	}
	source := data.DataSyncTask{Exchange: request.Exchange, Symbol: request.Symbol, Interval: request.Interval}
	for _, openTime := range request.OpenTimes {
		index, issue, ok := repository.fakeMarketCandleInvalidIssueIndex(data.QuarantineMarketCandleInvalidIssuesRequest{
			Exchange: request.Exchange,
			Symbol:   request.Symbol,
			Interval: request.Interval,
		}, openTime.UTC())
		if !ok {
			return data.DataSyncGapRepairResult{}, data.ErrNotFound
		}
		if !data.IsRepairableCandleIssueCode(issue.Code) {
			continue
		}
		candle := repository.candles[index]
		from := candle.OpenTime.UTC()
		window := data.CandleGap{From: from, To: from.Add(duration), MissingCandles: 1}
		if repository.fakeRepairTaskExists(source, window) {
			result.SkippedExisting += 1
			continue
		}
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		startTime := window.From
		endTime := window.To
		repairTask := data.DataSyncTask{
			ID:              "dst_market_invalid_repair_" + strconv.Itoa(len(repository.tasks)+1),
			Exchange:        request.Exchange,
			Symbol:          request.Symbol,
			Interval:        request.Interval,
			StartTime:       &startTime,
			EndTime:         &endTime,
			RequestID:       request.RequestID,
			SyncEnabled:     true,
			RealtimeEnabled: false,
			Status:          data.TaskStatusPending,
			DataHealth:      data.DataSyncHealthSyncing,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		repository.tasks = append(repository.tasks, repairTask)
		result.CreatedTasks = append(result.CreatedTasks, repairTask)
	}
	return result, nil
}

func (repository *fakeRepository) QuarantineMarketCandleInvalidIssues(
	_ context.Context,
	request data.QuarantineMarketCandleInvalidIssuesRequest,
) (data.MarketCandleQuarantineResult, error) {
	result := data.MarketCandleQuarantineResult{
		Quarantined:     []data.MarketCandleQuarantineRecord{},
		TotalCount:      len(request.OpenTimes),
		QuarantineLimit: data.MaxMarketCandleInvalidIssueScanLimit,
	}
	for _, openTime := range request.OpenTimes {
		index, issue, ok := repository.fakeMarketCandleInvalidIssueIndex(request, openTime.UTC())
		if !ok {
			return data.MarketCandleQuarantineResult{}, data.ErrNotFound
		}
		if issue.Code != data.CandleIssueInvalidOpenTime {
			result.SkippedNonQuarantinable++
			continue
		}
		candle := repository.candles[index]
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		result.Quarantined = append(result.Quarantined, data.MarketCandleQuarantineRecord{
			Exchange:      candle.Exchange,
			Symbol:        candle.Symbol,
			Interval:      candle.Interval,
			OpenTime:      candle.OpenTime,
			CloseTime:     candle.CloseTime,
			Reason:        issue.Code,
			Message:       issue.Message,
			QuarantinedAt: now,
		})
		repository.candles = append(repository.candles[:index], repository.candles[index+1:]...)
	}
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

func (repository *fakeRepository) fakeMarketCandleInvalidIssueWindow(
	request data.RepairMarketCandleInvalidIssuesRequest,
	openTime time.Time,
	duration time.Duration,
) (data.CandleGap, bool) {
	index, issue, ok := repository.fakeMarketCandleInvalidIssueIndex(data.QuarantineMarketCandleInvalidIssuesRequest{
		Exchange: request.Exchange,
		Symbol:   request.Symbol,
		Interval: request.Interval,
	}, openTime)
	if ok && data.IsRepairableCandleIssueCode(issue.Code) {
		candle := repository.candles[index]
		from := candle.OpenTime.UTC()
		return data.CandleGap{From: from, To: from.Add(duration), MissingCandles: 1}, true
	}
	return data.CandleGap{}, false
}

func (repository *fakeRepository) fakeMarketCandleInvalidIssueIndex(
	request data.QuarantineMarketCandleInvalidIssuesRequest,
	openTime time.Time,
) (int, data.CandleIssue, bool) {
	for index, candle := range repository.candles {
		if candle.Exchange != request.Exchange || candle.Symbol != request.Symbol || candle.Interval != request.Interval {
			continue
		}
		if !candle.OpenTime.Equal(openTime) {
			continue
		}
		issue := data.DetectCandleIssue(candle)
		if issue == nil {
			continue
		}
		return index, *issue, true
	}
	return 0, data.CandleIssue{}, false
}
