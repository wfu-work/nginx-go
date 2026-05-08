package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/middlewares"
)

type SettingRouter struct{}

func (s *SettingRouter) InitSettingRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("settings").Use(middlewares.ApiLogger())
	group := router.Group("settings")
	{
		group.GET("list", settingApi.List)
	}
	{
		groupLogger.POST("", settingApi.Save)
		groupLogger.DELETE(":guid", settingApi.Delete)
	}
}
