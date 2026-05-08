package routers

import "github.com/gin-gonic/gin"

type LogRouter struct{}

func (s *LogRouter) InitLogRouter(router *gin.RouterGroup) {
	group := router.Group("logs")
	{
		group.GET("access", logApi.Access)
		group.GET("access/list", logApi.Access)
		group.GET("error", logApi.Error)
		group.GET("error/list", logApi.Error)
		group.GET("audit/list", logApi.Audit)
	}
}
