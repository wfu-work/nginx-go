package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

type NginxInstance struct {
	commonDomains.BaseDataEntity
	Name            string `gorm:"column:name;size:100;index" json:"name"`
	NodeGuid        string `gorm:"column:node_guid;size:50;index" json:"nodeGuid"`
	Mode            string `gorm:"column:mode;size:30" json:"mode"`
	Host            string `gorm:"column:host;size:255" json:"host"`
	ServiceName     string `gorm:"column:service_name;size:100" json:"serviceName"`
	Bin             string `gorm:"column:bin;size:500" json:"bin"`
	Systemctl       string `gorm:"column:systemctl;size:500" json:"systemctl"`
	MainConfig      string `gorm:"column:main_config;size:500" json:"mainConfig"`
	ManagedConfig   string `gorm:"column:managed_config;size:500" json:"managedConfig"`
	DockerContainer string `gorm:"column:docker_container;size:255" json:"dockerContainer"`
	AccessLog       string `gorm:"column:access_log;size:500" json:"accessLog"`
	ErrorLog        string `gorm:"column:error_log;size:500" json:"errorLog"`
	StubStatusURL   string `gorm:"column:stub_status_url;size:500" json:"stubStatusUrl"`
	Enabled         bool   `gorm:"column:enabled;index" json:"enabled"`
	Description     string `gorm:"column:description;size:500" json:"description"`
}

func (NginxInstance) TableName() string {
	return "nginx_instances"
}
