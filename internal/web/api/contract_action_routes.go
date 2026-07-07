package api

import "net/http"

func addNotificationChannelContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/system/notifications/channels", http.MethodGet, operation(
		"system", "listNotificationChannels", "List notification channels", http.StatusOK, arraySchema(schemaRef("NotificationChannel")),
	))
	addOperation(paths, "/api/system/notifications/channels", http.MethodPost, operation(
		"system", "createNotificationChannel", "Create a notification channel", http.StatusCreated, schemaRef("NotificationChannel"),
		withCSRF(), withRequest(schemaRef("CreateNotificationChannel")), withErrors(http.StatusBadRequest),
	))
	addOperation(paths, "/api/system/notifications/channels/{id}", http.MethodPut, operation(
		"system", "updateNotificationChannel", "Update a notification channel", http.StatusOK, schemaRef("NotificationChannel"),
		withCSRF(), withParameters(pathParam("id", "Notification channel id")), withRequest(schemaRef("CreateNotificationChannel")), withErrors(http.StatusBadRequest, http.StatusNotFound),
	))
	addOperation(paths, "/api/system/notifications/channels/{id}", http.MethodDelete, operation(
		"system", "deleteNotificationChannel", "Delete a notification channel", http.StatusNoContent, nil,
		withCSRF(), withParameters(pathParam("id", "Notification channel id")), withErrors(http.StatusNotFound),
	))
	addSystemActionContractPaths(paths, "/api/system/notifications/channels/{id}",
		"setNotificationChannel", "Set notification channel", "NotificationChannel", "Notification channel id", "enable", "disable")
}

func addSystemActionContractPaths(
	paths map[string]apiPathItem,
	basePath string,
	operationPrefix string,
	summaryPrefix string,
	schemaName string,
	pathDescription string,
	actions ...string,
) {
	for _, action := range actions {
		addOperation(paths, basePath+"/"+action, http.MethodPost, operation(
			"system", operationPrefix+titleWord(action), summaryPrefix+" "+action, http.StatusOK, schemaRef(schemaName),
			withCSRF(), withParameters(pathParam("id", pathDescription)), withErrors(http.StatusNotFound),
		))
	}
}
