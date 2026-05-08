package routers

import "github.com/gin-gonic/gin"

type MetricRouter struct{}

func (s *MetricRouter) InitMetricRouter(router *gin.RouterGroup) {
	group := router.Group("metrics")
	{
		group.GET("summary", metricApi.Summary)
		group.GET("nginx", metricApi.StubStatus)
		group.GET("stub-status", metricApi.StubStatus)
		group.GET("process", metricApi.Process)
		group.GET("samples/list", metricApi.Samples)
	}
}
