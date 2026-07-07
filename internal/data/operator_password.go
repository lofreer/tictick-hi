package data

import (
	"errors"
	"fmt"
	"unicode"
)

const MinOperatorPasswordLength = 8

func ValidateOperatorPassword(password string) error {
	if len([]rune(password)) < MinOperatorPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinOperatorPasswordLength)
	}
	hasLetter := false
	hasDigit := false
	for _, character := range password {
		if unicode.IsLetter(character) {
			hasLetter = true
		}
		if unicode.IsDigit(character) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password must include at least one letter and one number")
	}
	return nil
}
