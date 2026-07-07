package data

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const MinOperatorPasswordLength = 8

const OperatorPasswordHistoryLimit = 5

var commonOperatorPasswords = map[string]struct{}{
	"admin123":    {},
	"admin1234":   {},
	"changeme1":   {},
	"changeme123": {},
	"letmein1":    {},
	"password1":   {},
	"password123": {},
	"qwerty123":   {},
	"secret123":   {},
	"secret1234":  {},
	"tictick123":  {},
	"tictickhi1":  {},
}

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
	if _, ok := commonOperatorPasswords[strings.ToLower(strings.TrimSpace(password))]; ok {
		return errors.New("password is too common")
	}
	return nil
}

func ValidateOperatorPasswordForUsername(username string, password string) error {
	if err := ValidateOperatorPassword(password); err != nil {
		return err
	}
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	if len([]rune(normalizedUsername)) >= 3 &&
		strings.Contains(strings.ToLower(password), normalizedUsername) {
		return errors.New("password must not include the username")
	}
	return nil
}
