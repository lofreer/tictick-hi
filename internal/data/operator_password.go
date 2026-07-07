package data

import "fmt"

const MinOperatorPasswordLength = 8

func ValidateOperatorPassword(password string) error {
	if len([]rune(password)) < MinOperatorPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinOperatorPasswordLength)
	}
	return nil
}
