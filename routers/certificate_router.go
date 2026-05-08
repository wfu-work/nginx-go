package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type CertificateRouter struct{}

func (s *CertificateRouter) InitCertificateRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("certificates").Use(middlewares.ApiLogger())
	group := router.Group("certificates")
	{
		group.GET("list", certApi.List)
		group.GET(":guid", certApi.Get)
	}
	{
		groupLogger.POST("", certApi.Create)
		groupLogger.PUT(":guid", certApi.Update)
		groupLogger.DELETE(":guid", certApi.Delete)
	}
}
