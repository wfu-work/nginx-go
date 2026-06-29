package routers

import "github.com/gin-gonic/gin"

type EventRouter struct{}

func (s *EventRouter) InitEventRouter(router *gin.RouterGroup) {
	group := router.Group("events")
	{
		group.GET("stream", eventApi.Stream)
		group.GET("ws", eventApi.WebSocket)
		group.GET("notifications/list", eventApi.NotificationList)
		group.POST("notifications/:guid/read", eventApi.MarkNotificationRead)
		group.POST("notifications/read-all", eventApi.MarkAllNotificationsRead)
	}
	router.GET("ws", eventApi.WebSocket)
}
