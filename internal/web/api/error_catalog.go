package api

import "net/http"

type apiErrorCode string

const (
	apiErrorInvalidRequest              apiErrorCode = "invalid_request"
	apiErrorUnauthorized                apiErrorCode = "unauthorized"
	apiErrorForbidden                   apiErrorCode = "forbidden"
	apiErrorCSRFRequired                apiErrorCode = "csrf_required"
	apiErrorCSRFInvalid                 apiErrorCode = "csrf_invalid"
	apiErrorNotFound                    apiErrorCode = "not_found"
	apiErrorMethodNotAllowed            apiErrorCode = "method_not_allowed"
	apiErrorConflict                    apiErrorCode = "conflict"
	apiErrorInvalidState                apiErrorCode = "invalid_state"
	apiErrorMarketInstrumentNotActive   apiErrorCode = "market_instrument_not_active"
	apiErrorDataSyncRetryRequiresFailed apiErrorCode = "data_sync_retry_requires_failed"
	apiErrorDataSyncCommandInvalidState apiErrorCode = "data_sync_command_invalid_state"
	apiErrorTooManyRequests             apiErrorCode = "too_many_requests"
	apiErrorInternal                    apiErrorCode = "internal_error"
	apiErrorRequestFailed               apiErrorCode = "request_failed"
)

type apiErrorDefinition struct {
	Code        string `json:"code"`
	HTTPStatus  int    `json:"httpStatus"`
	Description string `json:"description"`
	Retryable   bool   `json:"retryable"`
}

var apiErrorDefinitions = []apiErrorDefinition{
	{Code: string(apiErrorInvalidRequest), HTTPStatus: http.StatusBadRequest, Description: "The request payload, query, or route parameters are invalid."},
	{Code: string(apiErrorUnauthorized), HTTPStatus: http.StatusUnauthorized, Description: "The operator session is missing, expired, or invalid."},
	{Code: string(apiErrorForbidden), HTTPStatus: http.StatusForbidden, Description: "The authenticated operator is not allowed to perform this action."},
	{Code: string(apiErrorCSRFRequired), HTTPStatus: http.StatusForbidden, Description: "A CSRF token is required for this write request."},
	{Code: string(apiErrorCSRFInvalid), HTTPStatus: http.StatusForbidden, Description: "The supplied CSRF token does not match the session token."},
	{Code: string(apiErrorNotFound), HTTPStatus: http.StatusNotFound, Description: "The requested API resource does not exist."},
	{Code: string(apiErrorMethodNotAllowed), HTTPStatus: http.StatusMethodNotAllowed, Description: "The route exists but does not accept this HTTP method."},
	{Code: string(apiErrorConflict), HTTPStatus: http.StatusConflict, Description: "The request conflicts with the current resource state."},
	{Code: string(apiErrorInvalidState), HTTPStatus: http.StatusConflict, Description: "The resource state does not allow the requested transition."},
	{Code: string(apiErrorMarketInstrumentNotActive), HTTPStatus: http.StatusBadRequest, Description: "The requested market instrument is missing or not active in the local catalog."},
	{Code: string(apiErrorDataSyncRetryRequiresFailed), HTTPStatus: http.StatusConflict, Description: "The data sync task must be failed before it can be retried."},
	{Code: string(apiErrorDataSyncCommandInvalidState), HTTPStatus: http.StatusConflict, Description: "The data sync task state does not allow the requested command."},
	{Code: string(apiErrorTooManyRequests), HTTPStatus: http.StatusTooManyRequests, Description: "The caller has exceeded the accepted request rate.", Retryable: true},
	{Code: string(apiErrorInternal), HTTPStatus: http.StatusInternalServerError, Description: "The server failed while processing the request.", Retryable: true},
	{Code: string(apiErrorRequestFailed), HTTPStatus: http.StatusBadRequest, Description: "The request failed but does not map to a more specific API error code."},
}

func apiErrorCatalog() []apiErrorDefinition {
	catalog := make([]apiErrorDefinition, len(apiErrorDefinitions))
	copy(catalog, apiErrorDefinitions)
	return catalog
}

func apiErrorCodeKnown(code apiErrorCode) bool {
	for _, definition := range apiErrorDefinitions {
		if definition.Code == string(code) {
			return true
		}
	}
	return false
}

func apiErrorCodes() []string {
	codes := make([]string, 0, len(apiErrorDefinitions))
	for _, definition := range apiErrorDefinitions {
		codes = append(codes, definition.Code)
	}
	return codes
}

func apiErrorCodeSchema() map[string]any {
	return map[string]any{
		"type": "string",
		"enum": apiErrorCodes(),
	}
}

func apiErrorCodesForHTTPStatus(status int) []string {
	codes := []string{}
	for _, definition := range apiErrorDefinitions {
		if definition.HTTPStatus == status {
			codes = append(codes, definition.Code)
		}
	}
	return codes
}
