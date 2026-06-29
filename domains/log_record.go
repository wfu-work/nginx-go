package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

// AccessLogRecord stores one parsed nginx access log line for later filtering and aggregation.
type AccessLogRecord struct {
	commonDomains.BaseDataEntity
	InstanceGuid  string `gorm:"column:instance_guid;size:50;index" json:"instanceGuid"`
	RemoteAddr    string `gorm:"column:remote_addr;size:100;index" json:"remoteAddr"`
	TimeLocal     string `gorm:"column:time_local;size:100" json:"timeLocal"`
	Method        string `gorm:"column:method;size:20;index" json:"method"`
	Path          string `gorm:"column:path;size:1000;index" json:"path"`
	Protocol      string `gorm:"column:protocol;size:30" json:"protocol"`
	Status        int    `gorm:"column:status;index" json:"status"`
	BodyBytesSent int64  `gorm:"column:body_bytes_sent" json:"bodyBytesSent"`
	Referer       string `gorm:"column:referer;size:1000" json:"referer"`
	UserAgent     string `gorm:"column:user_agent;size:1000" json:"userAgent"`
	RawLine       string `gorm:"column:raw_line;type:text" json:"rawLine"`
	LineHash      string `gorm:"column:line_hash;size:64;uniqueIndex" json:"lineHash"`
}

func (AccessLogRecord) TableName() string {
	return "nginx_access_log_records"
}

// ErrorLogRecord stores one parsed nginx error log line for alerting and troubleshooting.
type ErrorLogRecord struct {
	commonDomains.BaseDataEntity
	InstanceGuid string `gorm:"column:instance_guid;size:50;index" json:"instanceGuid"`
	TimeLocal    string `gorm:"column:time_local;size:100" json:"timeLocal"`
	Level        string `gorm:"column:level;size:30;index" json:"level"`
	Message      string `gorm:"column:message;type:text" json:"message"`
	RawLine      string `gorm:"column:raw_line;type:text" json:"rawLine"`
	LineHash     string `gorm:"column:line_hash;size:64;uniqueIndex" json:"lineHash"`
}

func (ErrorLogRecord) TableName() string {
	return "nginx_error_log_records"
}
