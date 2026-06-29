package apis

import (
	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type LogApi struct{}

// Access returns raw access log lines from the configured nginx instance.
func (LogApi) Access(c *gin.Context) {
	result, err := logService.Access(queryParams(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Error returns raw error log lines from the configured nginx instance.
func (LogApi) Error(c *gin.Context) {
	result, err := logService.Error(queryParams(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// AccessRecords returns parsed access log records saved by the log sync endpoint.
func (LogApi) AccessRecords(c *gin.Context) {
	params := queryParams(c)
	items, total, err := logService.AccessRecords(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

// ErrorRecords returns parsed error log records saved by the log sync endpoint.
func (LogApi) ErrorRecords(c *gin.Context) {
	params := queryParams(c)
	items, total, err := logService.ErrorRecords(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

// Sync parses recent nginx log lines into structured database records.
func (LogApi) Sync(c *gin.Context) {
	result, err := logService.Sync(queryParams(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Audit returns operation and publish audit records.
func (LogApi) Audit(c *gin.Context) {
	params := queryParams(c)
	items, total, err := auditService.List(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}
