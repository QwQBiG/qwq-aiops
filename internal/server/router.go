package server

import (
	"io/fs"
	"net/http"
	"qwq/internal/deployment"
	"sync"
)

var deploymentOnce sync.Once

// NewHandler creates an independent router without starting background work.
func NewHandler() http.Handler {
	deploymentOnce.Do(func() {
		deploymentService = deployment.NewIntegrationService(GetDefaultFrontendManagerAdapter())
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleLiveness)
	routes := map[string]http.HandlerFunc{
		"/api/logs":                  handleLogs,
		"/api/stats":                 handleStats,
		"/api/trigger":               handleTrigger,
		"/api/containers":            handleContainers,
		"/api/container/action":      handleContainerAction,
		"/api/websites/":             handleWebsiteDetail,
		"/api/websites":              handleWebsites,
		"/api/users/":                handleUserDetail,
		"/api/users":                 handleUsers,
		"/api/roles/":                handleRoleDetail,
		"/api/roles":                 handleRoles,
		"/api/permissions":           handlePermissions,
		"/api/files/list":            handleFileList,
		"/api/files/content":         handleFileContent,
		"/api/files/save":            handleFileSave,
		"/api/files/action":          handleFileAction,
		"/api/appstore/templates":    handleAppStoreTemplates,
		"/api/appstore/instances":    handleAppStoreInstances,
		"/api/databases/connections": handleDatabaseConnections,
		"/api/deployment/validate":   handleDeploymentValidation,
		"/api/deployment/repair":     handleDeploymentRepair,
		"/api/deployment/status":     handleDeploymentStatus,
		"/api/deployment/workflow":   handleDeploymentWorkflow,
		"/api/health":                handleHealthCheck,
		"/ws/chat":                   handleWSChat,
	}
	for route, handler := range routes {
		mux.HandleFunc(route, basicAuth(handler))
	}
	missing := basicAuth(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusNotFound, "Endpoint not found", nil)
	})
	for _, prefix := range []string{"/api", "/api/", "/ws", "/ws/"} {
		mux.HandleFunc(prefix, missing)
	}
	assets, _ := fs.Sub(frontendDist, "dist")
	mux.Handle("/", basicAuth(newFrontendHandler(assets).ServeHTTP))
	return mux
}
