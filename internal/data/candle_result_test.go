package data

import "testing"

func TestValidateStrategyCandleResult(t *testing.T) {
	cases := []struct {
		name    string
		result  CandleResult
		wantErr bool
	}{
		{
			name: "healthy",
			result: CandleResult{
				Health: CandleHealthOK,
			},
		},
		{
			name: "gap",
			result: CandleResult{
				Health: CandleHealthGap,
			},
			wantErr: true,
		},
		{
			name: "insufficient",
			result: CandleResult{
				Health: CandleHealthInsufficient,
			},
			wantErr: true,
		},
		{
			name: "invalid",
			result: CandleResult{
				Health: CandleHealthInvalid,
			},
			wantErr: true,
		},
		{
			name: "limited coverage",
			result: CandleResult{
				Health: CandleHealthOK,
				Coverage: CandleCoverage{
					RequestedLimit:      1000,
					ReturnedCandles:     85,
					LimitedByBaseWindow: true,
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStrategyCandleResult(tc.result)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
