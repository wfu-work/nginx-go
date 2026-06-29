package routers

import "github.com/gin-gonic/gin"

type AgentRouter struct{}

func (s *AgentRouter) InitAgentRouter(router *gin.RouterGroup) {
	group := router.Group("agent")
	{
		group.POST("register", agentApi.Register)
		group.POST("heartbeat", agentApi.Heartbeat)
		group.GET("tasks/poll", agentApi.Poll)
		group.POST("tasks/:guid/complete", agentApi.Complete)
	}
}
