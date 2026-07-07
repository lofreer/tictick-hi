package data

import (
	"fmt"
	"strings"
)

const (
	OperatorRoleAdmin    = "admin"
	OperatorRoleOperator = "operator"
)

func NormalizeCreateOperatorRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return OperatorRoleOperator
	}
	return role
}

func ValidateOperatorRole(role string) error {
	switch role {
	case OperatorRoleAdmin, OperatorRoleOperator:
		return nil
	default:
		return fmt.Errorf("operator role must be admin or operator")
	}
}

func (operator Operator) IsAdmin() bool {
	return operator.Role == OperatorRoleAdmin
}
