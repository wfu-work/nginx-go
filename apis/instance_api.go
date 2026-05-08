package apis

import (
	"nginx-go/domains"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type InstanceApi struct{}

func (InstanceApi) List(c *gin.Context) {
	params := queryParams(c)
	items, total, err := instanceService.List(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

func (InstanceApi) Create(c *gin.Context) {
	var instance domains.NginxInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := instanceService.Create(instance); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

func (InstanceApi) Get(c *gin.Context) {
	result, err := instanceService.Get(c.Param("guid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (InstanceApi) Update(c *gin.Context) {
	var instance domains.NginxInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := instanceService.Update(c.Param("guid"), instance); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

func (InstanceApi) Delete(c *gin.Context) {
	if err := instanceService.Delete(c.Param("guid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}
