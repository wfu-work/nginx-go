package apis

import (
	"nginx-go/domains"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type SettingApi struct{}

func (SettingApi) List(c *gin.Context) {
	params := queryParams(c)
	items, total, err := settingService.List(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

func (SettingApi) Save(c *gin.Context) {
	var setting domains.Setting
	if err := c.ShouldBindJSON(&setting); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := settingService.Save(setting); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

func (SettingApi) Delete(c *gin.Context) {
	if err := settingService.Delete(c.Param("guid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}
