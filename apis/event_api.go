package apis

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type EventApi struct{}

// Stream pushes periodic metric snapshots through server-sent events.
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

// NotificationList returns paginated header notifications.
func (EventApi) NotificationList(c *gin.Context) {
	params := queryParams(c)
	items, total, err := eventService.List(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

// MarkNotificationRead marks one notification as read.
func (EventApi) MarkNotificationRead(c *gin.Context) {
	if err := eventService.MarkRead(c.Param("guid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// MarkAllNotificationsRead marks all notifications as read.
func (EventApi) MarkAllNotificationsRead(c *gin.Context) {
	if err := eventService.MarkAllRead(); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// WebSocket streams notification events to the frontend header message widget.
func (EventApi) WebSocket(c *gin.Context) {
	conn, rw, err := upgradeWebSocket(c.Writer, c.Request)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	defer conn.Close()
	events, cancel := eventService.Subscribe()
	defer cancel()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case payload, ok := <-events:
			if !ok {
				return
			}
			if _, err := rw.Write(websocketTextFrame(payload)); err != nil {
				return
			}
			if err := rw.Flush(); err != nil {
				return
			}
		}
	}
}

func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, nil, errors.New("missing websocket upgrade header")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, nil, errors.New("missing Sec-WebSocket-Key")
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("websocket hijacking is not supported")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, err
	}
	accept := websocketAcceptKey(key)
	_, err = fmt.Fprintf(
		rw,
		"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		accept,
	)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return conn, rw, nil
}

func websocketAcceptKey(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func websocketTextFrame(payload []byte) []byte {
	header := []byte{0x81}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, byte(length))
	case length <= 65535:
		header = append(header, 126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
	default:
		header = append(header, 127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	}
	return append(header, payload...)
}
