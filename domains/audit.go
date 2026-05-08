package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

const (
	AuditActionNginxOperation = "nginx_operation"
	AuditActionConfigPublish  = "config_publish"
	AuditActionConfigRollback = "config_rollback"
)

type AuditLog struct {
	commonDomains.BaseDataEntity
	Action       string `gorm:"column:action;size:50;index" json:"action"`
	ResourceType string `gorm:"column:resource_type;size:50;index" json:"resourceType"`
	ResourceGuid string `gorm:"column:resource_guid;size:50;index" json:"resourceGuid"`
	Success      bool   `gorm:"column:success" json:"success"`
	Status       string `gorm:"column:status;size:30;index" json:"status"`
	Message      string `gorm:"column:message;size:500" json:"message"`
	Reason       string `gorm:"column:reason;size:500" json:"reason"`
	Detail       string `gorm:"column:detail;type:text" json:"detail"`
}

func (AuditLog) TableName() string {
	return "nginx_audit_logs"
}
