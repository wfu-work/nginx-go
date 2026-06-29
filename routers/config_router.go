package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type ConfigRouter struct{}

// InitConfigRouter registers config render, validate, diff, publish, rollback, and history endpoints.
func (s *ConfigRouter) InitConfigRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("configs").Use(middlewares.ApiLogger())
	group := router.Group("configs")
	{
		group.GET("versions/list", configApi.VersionList)
		group.GET("versions/:guid", configApi.VersionGet)
		group.GET("tasks/list", configApi.TaskList)
	}
	{
		groupLogger.POST("render", configApi.Render)
		groupLogger.POST("validate", configApi.Validate)
		groupLogger.GET("diff", configApi.Diff)
		groupLogger.POST("diff", configApi.Diff)
		groupLogger.POST("publish", configApi.Publish)
		groupLogger.POST("rollback", configApi.Rollback)
	}
}
