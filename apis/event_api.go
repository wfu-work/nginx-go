package apis

import (
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

type EventApi struct{}

func (EventApi) Stream(c *gin.Context) {
	instanceGuid := c.Query("instanceGuid")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		result, err := metricService.Summary(instanceGuid)
		if err != nil {
			c.SSEvent("error", gin.H{"message": err.Error(), "time": time.Now().UnixMilli()})
		} else {
			c.SSEvent("metrics", result)
		}
		select {
		case <-c.Request.Context().Done():
			return false
		case <-ticker.C:
			return true
		}
	})
}
