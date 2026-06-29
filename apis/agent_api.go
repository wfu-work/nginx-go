package apis

import (
	"nginx-go/services"

	"github.com/gin-gonic/gin"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
)

type AgentApi struct{}

func (AgentApi) Register(c *gin.Context) {
	if !validAgentToken(c) {
		response.FailWithMessage("invalid agent token", c)
		return
	}
	var req services.AgentRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	node, err := agentService.Register(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(node, c)
}

func (AgentApi) Heartbeat(c *gin.Context) {
	if !validAgentToken(c) {
		response.FailWithMessage("invalid agent token", c)
		return
	}
	var req services.AgentHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	node, err := agentService.Heartbeat(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(node, c)
}

func (AgentApi) Poll(c *gin.Context) {
	if !validAgentToken(c) {
		response.FailWithMessage("invalid agent token", c)
		return
	}
	tasks, err := agentService.Poll(c.Query("nodeGuid"), c.Query("agentId"), 5)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(tasks, c)
}

func (AgentApi) Complete(c *gin.Context) {
	if !validAgentToken(c) {
		response.FailWithMessage("invalid agent token", c)
		return
	}
	var req services.AgentTaskCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	task, err := agentService.Complete(c.Param("guid"), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(task, c)
}

func validAgentToken(c *gin.Context) bool {
	if global.NAV_VIPER == nil {
		return true
	}
	expected := global.NAV_VIPER.GetString("agent.shared-token")
	if expected == "" {
		return true
	}
	return c.GetHeader("X-Agent-Token") == expected
}
