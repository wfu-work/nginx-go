package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type InstanceRouter struct{}

func (s *InstanceRouter) InitInstanceRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("nginx/instances").Use(middlewares.ApiLogger())
	group := router.Group("nginx/instances")
	{
		group.GET("list", instanceApi.List)
		group.GET(":guid", instanceApi.Get)
	}
	{
		groupLogger.POST("", instanceApi.Create)
		groupLogger.PUT(":guid", instanceApi.Update)
		groupLogger.DELETE(":guid", instanceApi.Delete)
	}
}
