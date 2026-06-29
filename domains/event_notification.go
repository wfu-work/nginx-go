package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

const (
	EventNotificationLevelInfo    = "info"
	EventNotificationLevelWarning = "warning"
	EventNotificationLevelError   = "error"
)

// EventNotification stores user-facing event messages for the header notification center.
type EventNotification struct {
	commonDomains.BaseDataEntity
	Title      string `gorm:"column:title;size:120;index" json:"title"`
	Content    string `gorm:"column:content;size:1000" json:"content"`
	Level      string `gorm:"column:level;size:30;index" json:"level"`
	Read       bool   `gorm:"column:is_read;index" json:"read"`
	SourceType string `gorm:"column:source_type;size:80;index" json:"sourceType"`
	SourceGuid string `gorm:"column:source_guid;size:80;index" json:"sourceGuid"`
	EventTime  int64  `gorm:"column:event_time;index" json:"eventTime"`
}

func (EventNotification) TableName() string {
	return "nginx_event_notifications"
}
