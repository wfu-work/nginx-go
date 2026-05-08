package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

const (
	NginxActionStatus  = "status"
	NginxActionRefresh = "refresh"
	NginxActionTest    = "test"
	NginxActionReload  = "reload"
	NginxActionRestart = "restart"
	NginxActionStart   = "start"
	NginxActionStop    = "stop"
)

type NginxOperation struct {
	commonDomains.BaseDataEntity
	InstanceGuid string `gorm:"column:instance_guid;size:50;index" json:"instanceGuid"`
	Action       string `gorm:"column:action;size:30;index" json:"action"`
	Success      bool   `gorm:"column:success" json:"success"`
	Status       string `gorm:"column:status;size:30;index" json:"status"`
	Message      string `gorm:"column:message;size:500" json:"message"`
	Command      string `gorm:"column:command;size:500" json:"command"`
	Output       string `gorm:"column:output;type:text" json:"output"`
	DurationMs   int64  `gorm:"column:duration_ms" json:"durationMs"`
	Reason       string `gorm:"column:reason;size:500" json:"reason"`
}

func (NginxOperation) TableName() string {
	return "nginx_operations"
}
