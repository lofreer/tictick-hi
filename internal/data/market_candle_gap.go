package data

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
