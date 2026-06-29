package apis

import (
	"nginx-go/domains"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type UpstreamApi struct{}

// List returns paginated upstream groups.
func (UpstreamApi) List(c *gin.Context) {
	params := queryParams(c)
	items, total, err := upstreamService.List(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

// Create stores an upstream group.
func (UpstreamApi) Create(c *gin.Context) {
	var upstream domains.Upstream
	if err := c.ShouldBindJSON(&upstream); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := upstreamService.Create(upstream); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// Get returns an upstream group with its servers.
func (UpstreamApi) Get(c *gin.Context) {
	result, err := upstreamService.Get(c.Param("guid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Health checks upstream server TCP connectivity.
func (UpstreamApi) Health(c *gin.Context) {
	result, err := upstreamService.Health(c.Param("guid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Update modifies an upstream group.
func (UpstreamApi) Update(c *gin.Context) {
	var upstream domains.Upstream
	if err := c.ShouldBindJSON(&upstream); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := upstreamService.Update(c.Param("guid"), upstream); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// Delete removes an upstream group.
func (UpstreamApi) Delete(c *gin.Context) {
	if err := upstreamService.Delete(c.Param("guid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// CreateServer stores an upstream server under a group.
func (UpstreamApi) CreateServer(c *gin.Context) {
	var server domains.UpstreamServer
	if err := c.ShouldBindJSON(&server); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if server.UpstreamGuid == "" {
		server.UpstreamGuid = c.Param("guid")
	}
	if err := upstreamService.CreateServer(server); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// UpdateServer modifies one upstream server.
func (UpstreamApi) UpdateServer(c *gin.Context) {
	var server domains.UpstreamServer
	if err := c.ShouldBindJSON(&server); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := upstreamService.UpdateServer(c.Param("serverGuid"), server); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// DeleteServer removes one upstream server.
func (UpstreamApi) DeleteServer(c *gin.Context) {
	if err := upstreamService.DeleteServer(c.Param("serverGuid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}
