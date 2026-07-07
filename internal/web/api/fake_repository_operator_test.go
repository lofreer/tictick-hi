package api

import (
	"context"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) ListOperators(context.Context) ([]data.Operator, error) {
	return append([]data.Operator(nil), repository.operators...), nil
}

func (repository *fakeRepository) CreateOperator(
	_ context.Context,
	request data.CreateOperator,
) (data.Operator, error) {
	if err := data.ValidateOperatorPasswordForUsername(request.Username, request.Password); err != nil {
		return data.Operator{}, err
	}
	role := data.NormalizeCreateOperatorRole(request.Role)
	if err := data.ValidateOperatorRole(role); err != nil {
		return data.Operator{}, err
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	operator := data.Operator{
		ID:        "op_" + request.Username,
		Username:  request.Username,
		Role:      role,
		Enabled:   request.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.operators = append(repository.operators, operator)
	repository.passwords[operator.ID] = request.Password
	return operator, nil
}

func (repository *fakeRepository) SetOperatorEnabled(
	_ context.Context,
	id string,
	enabled bool,
) (data.Operator, error) {
	for index := range repository.operators {
		if repository.operators[index].ID == id {
			if !enabled && repository.operators[index].Enabled {
				if repository.enabledOperatorCount() <= 1 {
					return data.Operator{}, data.OperatorLastEnabledError()
				}
				if repository.operators[index].Role == data.OperatorRoleAdmin && repository.enabledAdminOperatorCount() <= 1 {
					return data.Operator{}, data.OperatorLastAdminError()
				}
			}
			repository.operators[index].Enabled = enabled
			return repository.operators[index], nil
		}
	}
	return data.Operator{}, data.ErrNotFound
}

func (repository *fakeRepository) enabledOperatorCount() int {
	count := 0
	for _, operator := range repository.operators {
		if operator.Enabled {
			count++
		}
	}
	return count
}

func (repository *fakeRepository) enabledAdminOperatorCount() int {
	count := 0
	for _, operator := range repository.operators {
		if operator.Enabled && operator.Role == data.OperatorRoleAdmin {
			count++
		}
	}
	return count
}

func (repository *fakeRepository) AuthenticateOperator(
	_ context.Context,
	username string,
	password string,
) (data.Operator, error) {
	for _, operator := range repository.operators {
		if operator.Username == username && operator.Enabled && repository.passwords[operator.ID] == password {
			return operator, nil
		}
	}
	return data.Operator{}, data.ErrUnauthorized
}
