package apis

import (
	"nginx-go/domains"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type NodeApi struct{}

func (NodeApi) List(c *gin.Context) {
	params := queryParams(c)
	items, total, err := nodeService.List(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

func (NodeApi) Create(c *gin.Context) {
	var node domains.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := nodeService.Create(node); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

func (NodeApi) Get(c *gin.Context) {
	result, err := nodeService.Get(c.Param("guid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (NodeApi) Update(c *gin.Context) {
	var node domains.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := nodeService.Update(c.Param("guid"), node); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

func (NodeApi) Delete(c *gin.Context) {
	if err := nodeService.Delete(c.Param("guid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}
