package router

import (
	"path"

	"github.com/Im-oldxu/new-api/controller"
	"github.com/Im-oldxu/new-api/middleware"
	"github.com/Im-oldxu/new-api/service/authz"

	"github.com/gin-gonic/gin"
)

// registerAuthzRoutes mounts the authorization API under its own /authz
// namespace. GET /authz/catalog returns the permission schema (resources,
// actions, and role baselines) used by the client permission editor.
func registerAuthzRoutes(apiRouter *gin.RouterGroup) {
	authzRoute := apiRouter.Group("/authz")
	authzRoute.Use(middleware.AdminAuth())
	{
		authzRoute.GET("/catalog", controller.GetPermissionCatalog)
	}
}

func handlePermissionRoute(group *gin.RouterGroup, method, relativePath string, permission authz.Permission, handlers ...gin.HandlerFunc) {
	middleware.DeclareAccessTokenPermissionRoute(method, joinPaths(group.BasePath(), relativePath), permission)
	group.Handle(method, relativePath, append([]gin.HandlerFunc{middleware.RequirePermission(permission)}, handlers...)...)
}

func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}
	finalPath := path.Join(absolutePath, relativePath)
	if relativePath[len(relativePath)-1] == '/' && finalPath[len(finalPath)-1] != '/' {
		return finalPath + "/"
	}
	return finalPath
}
