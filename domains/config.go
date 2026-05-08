package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

const (
	ConfigVersionStatusRendered   = "rendered"
	ConfigVersionStatusValidated  = "validated"
	ConfigVersionStatusPublished  = "published"
	ConfigVersionStatusRolledBack = "rolled_back"
)

type ConfigVersion struct {
	commonDomains.BaseDataEntity
	SiteGuid     string `gorm:"column:site_guid;size:50;index" json:"siteGuid"`
	VersionNo    int64  `gorm:"column:version_no;index" json:"versionNo"`
	Status       string `gorm:"column:status;size:30;index" json:"status"`
	Config       string `gorm:"column:config;type:text" json:"config"`
	ValidateOK   bool   `gorm:"column:validate_ok" json:"validateOk"`
	ValidateMsg  string `gorm:"column:validate_msg;type:text" json:"validateMsg"`
	PublishedAt  int64  `gorm:"column:published_at" json:"publishedAt"`
	RollbackFrom string `gorm:"column:rollback_from;size:50" json:"rollbackFrom"`
	Reason       string `gorm:"column:reason;size:500" json:"reason"`
}

func (ConfigVersion) TableName() string {
	return "nginx_config_versions"
}

type PublishTask struct {
	commonDomains.BaseDataEntity
	VersionGuid   string `gorm:"column:version_guid;size:50;index" json:"versionGuid"`
	Action        string `gorm:"column:action;size:30;index" json:"action"`
	Success       bool   `gorm:"column:success" json:"success"`
	Status        string `gorm:"column:status;size:30;index" json:"status"`
	TargetPath    string `gorm:"column:target_path;size:500" json:"targetPath"`
	BackupPath    string `gorm:"column:backup_path;size:500" json:"backupPath"`
	Message       string `gorm:"column:message;type:text" json:"message"`
	OperationGuid string `gorm:"column:operation_guid;size:50" json:"operationGuid"`
	DurationMs    int64  `gorm:"column:duration_ms" json:"durationMs"`
	Reason        string `gorm:"column:reason;size:500" json:"reason"`
}

func (PublishTask) TableName() string {
	return "nginx_publish_tasks"
}
