package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

type MetricSample struct {
	commonDomains.BaseDataEntity
	Kind    string `gorm:"column:kind;size:50;index" json:"kind"`
	Status  string `gorm:"column:status;size:30;index" json:"status"`
	Payload string `gorm:"column:payload;type:text" json:"payload"`
	Message string `gorm:"column:message;size:500" json:"message"`
}

func (MetricSample) TableName() string {
	return "nginx_metric_samples"
}
