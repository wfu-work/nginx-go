package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type SiteRouter struct{}

func (s *SiteRouter) InitSiteRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("sites").Use(middlewares.ApiLogger())
	group := router.Group("sites")
	{
		group.GET("list", siteApi.List)
		group.GET(":guid", siteApi.Get)
	}
	{
		groupLogger.POST("", siteApi.Create)
		groupLogger.PUT(":guid", siteApi.Update)
		groupLogger.DELETE(":guid", siteApi.Delete)
		groupLogger.POST(":guid/enable", siteApi.Enable)
		groupLogger.POST(":guid/disable", siteApi.Disable)
		groupLogger.POST(":guid/locations", siteApi.CreateLocation)
		groupLogger.PUT(":guid/locations/:locationGuid", siteApi.UpdateLocation)
		groupLogger.DELETE(":guid/locations/:locationGuid", siteApi.DeleteLocation)
	}
}
