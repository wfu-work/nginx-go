package apis

import (
	"nginx-go/services"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type NginxApi struct{}

// Status returns current nginx runtime status for an instance.
func (NginxApi) Status(c *gin.Context) {
	result, err := nginxService.Status(c.Query("instanceGuid"))
	if err != nil {
		global.NAV_LOG.Error("get nginx status failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Refresh records a manual status refresh operation.
func (NginxApi) Refresh(c *gin.Context) {
	req := bindOperationRequest(c)
	result, err := nginxService.Refresh(req)
	if err != nil {
		global.NAV_LOG.Error("refresh nginx status failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Test validates nginx configuration with `nginx -t`.
func (NginxApi) Test(c *gin.Context) {
	req := bindOperationRequest(c)
	result, err := nginxService.Test(req)
	if err != nil {
		global.NAV_LOG.Error("test nginx config failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Reload validates then reloads nginx.
func (NginxApi) Reload(c *gin.Context) {
	req := bindOperationRequest(c)
	result, err := nginxService.Reload(req)
	if err != nil {
		global.NAV_LOG.Error("reload nginx failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Restart restarts nginx after confirmation validation.
func (NginxApi) Restart(c *gin.Context) {
	req := bindOperationRequest(c)
	result, err := nginxService.Restart(req)
	if err != nil {
		global.NAV_LOG.Error("restart nginx failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Start starts nginx for the selected instance.
func (NginxApi) Start(c *gin.Context) {
	req := bindOperationRequest(c)
	result, err := nginxService.Start(req)
	if err != nil {
		global.NAV_LOG.Error("start nginx failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Stop stops nginx after confirmation validation.
func (NginxApi) Stop(c *gin.Context) {
	req := bindOperationRequest(c)
	result, err := nginxService.Stop(req)
	if err != nil {
		global.NAV_LOG.Error("stop nginx failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// OperationList returns paginated nginx operation history.
func (NginxApi) OperationList(c *gin.Context) {
	params := queryParams(c)
	items, total, err := nginxService.OperationList(params)
	if err != nil {
		global.NAV_LOG.Error("list nginx operations failed", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{
		Data:  items,
		Total: total,
		Page:  commonUtils.Str2Int(params["page"]),
		Size:  commonUtils.Str2Int(params["size"]),
	}, c)
}

// OperationGet returns one nginx operation by guid.
func (NginxApi) OperationGet(c *gin.Context) {
	result, err := nginxService.OperationGet(c.Param("guid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func bindOperationRequest(c *gin.Context) services.OperationRequest {
	var req services.OperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = services.OperationRequest{}
	}
	if req.InstanceGuid == "" {
		req.InstanceGuid = c.Query("instanceGuid")
	}
	return req
}
