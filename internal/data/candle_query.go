package data

const (
	DefaultCandleLimit = 1000
	MaxCandleLimit     = 5000
)

func NormalizeCandleLimit(limit int) int {
	if limit <= 0 {
		return DefaultCandleLimit
	}
	if limit > MaxCandleLimit {
		return MaxCandleLimit
	}
	return limit
}
