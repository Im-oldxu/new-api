package router

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BusinessModule struct {
	Migrate  func(*gorm.DB) error
	Register func(*gin.RouterGroup)
}

func SetRouter(router *gin.Engine, assets WebAssets, modules ...BusinessModule) {
	SetApiRouter(router, modules...)
	frontendBaseUrl := os.Getenv("FRONTEND_BASE_URL")
	if common.IsMasterNode && frontendBaseUrl != "" {
		frontendBaseUrl = ""
		common.SysLog(common.LogText("FRONTEND_BASE_URL is ignored on master node"))
	}
	if frontendBaseUrl == "" {
		SetWebRouter(router, assets)
	} else {
		frontendBaseUrl = strings.TrimSuffix(frontendBaseUrl, "/")
		router.NoRoute(
			middleware.RouteTag("web"),
			middleware.AccessTokenAudit(),
			func(c *gin.Context) {
				if isUnregisteredAPIPath(c.Request.URL.Path) {
					respondNotFound(c)
					return
				}
				c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s%s", frontendBaseUrl, c.Request.RequestURI))
			},
		)
	}
}

func isUnregisteredAPIPath(path string) bool {
	namespace, rest, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	switch namespace {
	case "api", "v1", "v1beta", "pg", "mj":
		return true
	}
	return rest == "mj" || strings.HasPrefix(rest, "mj/")
}

func respondNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Not found"})
}
