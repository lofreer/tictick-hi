package data

import "testing"

func TestNormalizeCandleLimit(t *testing.T) {
	cases := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: DefaultCandleLimit},
		{name: "negative", limit: -1, want: DefaultCandleLimit},
		{name: "requested", limit: 250, want: 250},
		{name: "max", limit: MaxCandleLimit, want: MaxCandleLimit},
		{name: "oversized", limit: MaxCandleLimit + 1, want: MaxCandleLimit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeCandleLimit(tc.limit); got != tc.want {
				t.Fatalf("NormalizeCandleLimit(%d) = %d, want %d", tc.limit, got, tc.want)
			}
		})
	}
}
