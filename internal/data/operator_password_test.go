package data

import "testing"

func TestValidateOperatorPassword(t *testing.T) {
	if err := ValidateOperatorPassword("secret123"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
	if err := ValidateOperatorPassword("short"); err == nil {
		t.Fatal("short password was accepted")
	}
}
