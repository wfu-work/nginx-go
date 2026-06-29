package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type NginxRouter struct{}

// InitNginxRouter registers nginx process operation and operation history endpoints.
func (s *NginxRouter) InitNginxRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("nginx").Use(middlewares.ApiLogger())
	group := router.Group("nginx")
	{
		group.GET("status", nginxApi.Status)
		group.GET("operations/list", nginxApi.OperationList)
		group.GET("operations/:guid", nginxApi.OperationGet)
	}
	{
		groupLogger.POST("refresh", nginxApi.Refresh)
		groupLogger.POST("test", nginxApi.Test)
		groupLogger.POST("reload", nginxApi.Reload)
		groupLogger.POST("restart", nginxApi.Restart)
		groupLogger.POST("start", nginxApi.Start)
		groupLogger.POST("stop", nginxApi.Stop)
	}
}
