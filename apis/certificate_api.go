package apis

import (
	"nginx-go/domains"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type CertificateApi struct{}

func (CertificateApi) List(c *gin.Context) {
	params := queryParams(c)
	items, total, err := certService.List(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

func (CertificateApi) Create(c *gin.Context) {
	var cert domains.Certificate
	if err := c.ShouldBindJSON(&cert); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := certService.Create(cert); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

func (CertificateApi) Get(c *gin.Context) {
	result, err := certService.Get(c.Param("guid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

func (CertificateApi) Update(c *gin.Context) {
	var cert domains.Certificate
	if err := c.ShouldBindJSON(&cert); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := certService.Update(c.Param("guid"), cert); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

func (CertificateApi) Delete(c *gin.Context) {
	if err := certService.Delete(c.Param("guid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}
