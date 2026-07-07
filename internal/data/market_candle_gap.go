package data

import "time"

type MarketCandleGapScanQuery struct {
	Exchange string
	Symbol   string
	Interval string
	Limit    int
}

type MarketCandleGapScan struct {
	Exchange      string       `json:"exchange"`
	Symbol        string       `json:"symbol"`
	Interval      string       `json:"interval"`
	Window        CandleWindow `json:"window"`
	Gaps          []CandleGap  `json:"gaps"`
	Limited       bool         `json:"limited"`
	TotalCount    int          `json:"totalCount"`
	ReturnedCount int          `json:"returnedCount"`
}

type MarketCandleInvalidIssueScanQuery struct {
	Exchange string
	Symbol   string
	Interval string
	Limit    int
}

type MarketCandleInvalidIssueScan struct {
	Exchange      string        `json:"exchange"`
	Symbol        string        `json:"symbol"`
	Interval      string        `json:"interval"`
	Window        CandleWindow  `json:"window"`
	Issues        []CandleIssue `json:"issues"`
	Limited       bool          `json:"limited"`
	TotalCount    int           `json:"totalCount"`
	ReturnedCount int           `json:"returnedCount"`
}

type RepairMarketCandleGapRequest struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	RequestID string    `json:"-"`
}

type RepairMarketCandleGapWindow struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type RepairMarketCandleGapsRequest struct {
	Exchange  string                        `json:"exchange"`
	Symbol    string                        `json:"symbol"`
	Interval  string                        `json:"interval"`
	Gaps      []RepairMarketCandleGapWindow `json:"gaps"`
	RequestID string                        `json:"-"`
}

type RepairMarketCandleInvalidIssuesRequest struct {
	Exchange  string      `json:"exchange"`
	Symbol    string      `json:"symbol"`
	Interval  string      `json:"interval"`
	OpenTimes []time.Time `json:"openTimes"`
	RequestID string      `json:"-"`
}

type QuarantineMarketCandleInvalidIssuesRequest struct {
	Exchange  string      `json:"exchange"`
	Symbol    string      `json:"symbol"`
	Interval  string      `json:"interval"`
	OpenTimes []time.Time `json:"openTimes"`
}

type MarketCandleQuarantineRecord struct {
	Exchange      string    `json:"exchange"`
	Symbol        string    `json:"symbol"`
	Interval      string    `json:"interval"`
	OpenTime      time.Time `json:"openTime"`
	CloseTime     time.Time `json:"closeTime"`
	Reason        string    `json:"reason"`
	Message       string    `json:"message"`
	QuarantinedAt time.Time `json:"quarantinedAt"`
}

type MarketCandleQuarantineResult struct {
	Quarantined             []MarketCandleQuarantineRecord `json:"quarantined"`
	SkippedNonQuarantinable int                            `json:"skippedNonQuarantinable"`
	TotalCount              int                            `json:"totalCount"`
	QuarantineLimit         int                            `json:"quarantineLimit"`
}
