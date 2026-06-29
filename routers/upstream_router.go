package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type UpstreamRouter struct{}

// InitUpstreamRouter registers upstream group, server, and health endpoints.
func (s *UpstreamRouter) InitUpstreamRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("upstreams").Use(middlewares.ApiLogger())
	group := router.Group("upstreams")
	{
		group.GET("list", upstreamApi.List)
		group.GET(":guid", upstreamApi.Get)
		group.GET(":guid/health", upstreamApi.Health)
	}
	{
		groupLogger.POST("", upstreamApi.Create)
		groupLogger.PUT(":guid", upstreamApi.Update)
		groupLogger.DELETE(":guid", upstreamApi.Delete)
		groupLogger.POST(":guid/servers", upstreamApi.CreateServer)
		groupLogger.PUT(":guid/servers/:serverGuid", upstreamApi.UpdateServer)
		groupLogger.DELETE(":guid/servers/:serverGuid", upstreamApi.DeleteServer)
	}
}
