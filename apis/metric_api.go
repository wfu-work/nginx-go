package apis

import (
	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type MetricApi struct{}

// Summary returns combined nginx status and stub_status metrics.
func (MetricApi) Summary(c *gin.Context) {
	result, err := metricService.Summary(c.Query("instanceGuid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// StubStatus returns parsed nginx stub_status counters.
func (MetricApi) StubStatus(c *gin.Context) {
	result, err := metricService.StubStatus(c.Query("instanceGuid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Process returns detected nginx process metrics.
func (MetricApi) Process(c *gin.Context) {
	result, err := metricService.Process(c.Query("instanceGuid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Samples returns persisted metric samples.
func (MetricApi) Samples(c *gin.Context) {
	params := queryParams(c)
	items, total, err := metricService.Samples(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}
