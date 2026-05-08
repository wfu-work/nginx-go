package apis

import (
	"nginx-go/services"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type ConfigApi struct{}

func (ConfigApi) Render(c *gin.Context) {
	var req services.RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.SiteGuid = c.Query("siteGuid")
	}
	result, err := configService.Render(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (ConfigApi) Validate(c *gin.Context) {
	var req services.ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.SiteGuid = c.Query("siteGuid")
	}
	if req.InstanceGuid == "" {
		req.InstanceGuid = c.Query("instanceGuid")
	}
	if req.SiteGuid == "" {
		req.SiteGuid = c.Query("siteGuid")
	}
	result, err := configService.Validate(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (ConfigApi) Publish(c *gin.Context) {
	var req services.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.InstanceGuid == "" {
		req.InstanceGuid = c.Query("instanceGuid")
	}
	result, err := configService.Publish(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (ConfigApi) Rollback(c *gin.Context) {
	var req services.RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.InstanceGuid == "" {
		req.InstanceGuid = c.Query("instanceGuid")
	}
	result, err := configService.Rollback(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (ConfigApi) Diff(c *gin.Context) {
	var req services.DiffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.FromVersionGuid = c.Query("fromVersionGuid")
		req.ToVersionGuid = c.Query("toVersionGuid")
	}
	result, err := configService.Diff(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (ConfigApi) VersionList(c *gin.Context) {
	params := queryParams(c)
	items, total, err := configService.VersionList(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

func (ConfigApi) VersionGet(c *gin.Context) {
	result, err := configService.VersionGet(c.Param("guid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (ConfigApi) TaskList(c *gin.Context) {
	params := queryParams(c)
	items, total, err := configService.TaskList(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}
