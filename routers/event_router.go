package routers

import "github.com/gin-gonic/gin"

type EventRouter struct{}

func (s *EventRouter) InitEventRouter(router *gin.RouterGroup) {
	group := router.Group("events")
	{
		group.GET("stream", eventApi.Stream)
	}
}
