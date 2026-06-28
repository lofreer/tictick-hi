package data

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const candleCursorVersion = 1

type CandleCursor struct {
	Version  int       `json:"v"`
	Exchange string    `json:"exchange"`
	Symbol   string    `json:"symbol"`
	Interval string    `json:"interval"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	Limit    int       `json:"limit"`
}

func NewCandleCursor(query CandleQuery, from time.Time, to time.Time, limit int) CandleCursor {
	return CandleCursor{
		Version:  candleCursorVersion,
		Exchange: query.Exchange,
		Symbol:   query.Symbol,
		Interval: query.Interval,
		From:     from.UTC(),
		To:       to.UTC(),
		Limit:    NormalizeCandleLimit(limit),
	}
}

func EncodeCandleCursor(cursor CandleCursor) (string, error) {
	cursor.Version = candleCursorVersion
	cursor.From = cursor.From.UTC()
	cursor.To = cursor.To.UTC()
	cursor.Limit = NormalizeCandleLimit(cursor.Limit)
	if err := validateCandleCursor(cursor); err != nil {
		return "", err
	}
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func DecodeCandleCursor(token string) (CandleCursor, error) {
	if strings.TrimSpace(token) == "" {
		return CandleCursor{}, fmt.Errorf("cursor is required")
	}
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return CandleCursor{}, fmt.Errorf("cursor is invalid")
	}
	var cursor CandleCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return CandleCursor{}, fmt.Errorf("cursor is invalid")
	}
	cursor.From = cursor.From.UTC()
	cursor.To = cursor.To.UTC()
	if err := validateCandleCursor(cursor); err != nil {
		return CandleCursor{}, err
	}
	return cursor, nil
}

func (cursor CandleCursor) MatchesQuery(query CandleQuery) bool {
	return cursor.Exchange == query.Exchange &&
		cursor.Symbol == query.Symbol &&
		cursor.Interval == query.Interval
}

func validateCandleCursor(cursor CandleCursor) error {
	if cursor.Version != candleCursorVersion {
		return fmt.Errorf("cursor version is unsupported")
	}
	if cursor.Exchange == "" || cursor.Symbol == "" || cursor.Interval == "" {
		return fmt.Errorf("cursor context is incomplete")
	}
	if cursor.From.IsZero() || cursor.To.IsZero() {
		return fmt.Errorf("cursor window is incomplete")
	}
	if cursor.To.Before(cursor.From) {
		return fmt.Errorf("cursor window is invalid")
	}
	if cursor.Limit <= 0 || cursor.Limit > MaxCandleLimit {
		return fmt.Errorf("cursor limit is invalid")
	}
	if err := ValidateCandleQueryRange(CandleQuery{
		Exchange: cursor.Exchange,
		Symbol:   cursor.Symbol,
		Interval: cursor.Interval,
		From:     &cursor.From,
		To:       &cursor.To,
		Limit:    cursor.Limit,
	}); err != nil {
		return fmt.Errorf("cursor %w", err)
	}
	return nil
}
