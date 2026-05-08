package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

type Upstream struct {
	commonDomains.BaseDataEntity
	Name        string `gorm:"column:name;size:100;uniqueIndex" json:"name"`
	Method      string `gorm:"column:method;size:50" json:"method"`
	ExtraConfig string `gorm:"column:extra_config;type:text" json:"extraConfig"`
}

func (Upstream) TableName() string {
	return "nginx_upstreams"
}

type UpstreamServer struct {
	commonDomains.BaseDataEntity
	UpstreamGuid string `gorm:"column:upstream_guid;size:50;index" json:"upstreamGuid"`
	Address      string `gorm:"column:address;size:255" json:"address"`
	Weight       int    `gorm:"column:weight" json:"weight"`
	MaxFails     int    `gorm:"column:max_fails" json:"maxFails"`
	FailTimeout  string `gorm:"column:fail_timeout;size:50" json:"failTimeout"`
	Backup       bool   `gorm:"column:backup" json:"backup"`
	Down         bool   `gorm:"column:down" json:"down"`
	Sort         int    `gorm:"column:sort" json:"sort"`
}

func (UpstreamServer) TableName() string {
	return "nginx_upstream_servers"
}
