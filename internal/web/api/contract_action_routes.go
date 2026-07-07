package api

import "net/http"

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
