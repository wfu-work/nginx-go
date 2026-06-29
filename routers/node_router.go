package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type NodeRouter struct{}

func (s *NodeRouter) InitNodeRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("nodes").Use(middlewares.ApiLogger())
	group := router.Group("nodes")
	{
		group.GET("list", nodeApi.List)
		group.GET(":guid", nodeApi.Get)
	}
	{
		groupLogger.POST("", nodeApi.Create)
		groupLogger.PUT(":guid", nodeApi.Update)
		groupLogger.DELETE(":guid", nodeApi.Delete)
	}
}
