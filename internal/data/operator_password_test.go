package data

import "testing"

func TestValidateOperatorPassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{name: "valid", password: "secret123A"},
		{name: "too short", password: "short", wantError: true},
		{name: "missing digit", password: "password", wantError: true},
		{name: "missing letter", password: "12345678", wantError: true},
		{name: "common", password: "secret123", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateOperatorPassword(test.password)
			if test.wantError && err == nil {
				t.Fatal("password was accepted")
			}
			if !test.wantError && err != nil {
				t.Fatalf("password was rejected: %v", err)
			}
		})
	}
}
