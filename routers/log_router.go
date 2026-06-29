package routers

import "github.com/gin-gonic/gin"

type LogRouter struct{}

// InitLogRouter registers raw log, parsed log, sync, and audit query endpoints.
func (s *LogRouter) InitLogRouter(router *gin.RouterGroup) {
	group := router.Group("logs")
	{
		group.GET("access", logApi.Access)
		group.GET("access/list", logApi.Access)
		group.GET("access/records", logApi.AccessRecords)
		group.GET("error", logApi.Error)
		group.GET("error/list", logApi.Error)
		group.GET("error/records", logApi.ErrorRecords)
		group.POST("sync", logApi.Sync)
		group.GET("audit/list", logApi.Audit)
	}
}
