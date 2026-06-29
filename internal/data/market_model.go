package data

import "time"

type MarketInstrument struct {
	Exchange       string     `json:"exchange"`
	Symbol         string     `json:"symbol"`
	BaseAsset      string     `json:"baseAsset"`
	QuoteAsset     string     `json:"quoteAsset"`
	InstrumentType string     `json:"instrumentType"`
	Status         string     `json:"status"`
	ExchangeStatus string     `json:"exchangeStatus"`
	SearchPriority int        `json:"searchPriority"`
	SyncedAt       *time.Time `json:"syncedAt,omitempty"`
	CreatedAt      time.Time  `json:"-"`
	UpdatedAt      time.Time  `json:"-"`
}

type MarketInstrumentQuery struct {
	Exchange string
	Query    string
	Limit    int
	Status   string
}

type MarketInstrumentSyncResult struct {
	Exchange                string    `json:"exchange"`
	ActiveCount             int       `json:"activeCount"`
	InactiveCount           int       `json:"inactiveCount"`
	PausedDataSyncTaskCount int       `json:"pausedDataSyncTaskCount"`
	SyncedAt                time.Time `json:"syncedAt"`
}

type MarketInstrumentSyncStatus struct {
	Exchange      string     `json:"exchange"`
	LastAttemptAt time.Time  `json:"lastAttemptAt"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
