package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

type Site struct {
	commonDomains.BaseDataEntity
	Name        string `gorm:"column:name;size:100;index" json:"name"`
	ServerName  string `gorm:"column:server_name;size:255;index" json:"serverName"`
	Listen      string `gorm:"column:listen;size:50" json:"listen"`
	Enabled     bool   `gorm:"column:enabled;index" json:"enabled"`
	Root        string `gorm:"column:root;size:500" json:"root"`
	Index       string `gorm:"column:index_file;size:255" json:"index"`
	AccessLog   string `gorm:"column:access_log;size:500" json:"accessLog"`
	ErrorLog    string `gorm:"column:error_log;size:500" json:"errorLog"`
	CertificateGuid string `gorm:"column:certificate_guid;size:50;index" json:"certificateGuid"`
	SSL             bool   `gorm:"column:ssl" json:"ssl"`
	ExtraConfig string `gorm:"column:extra_config;type:text" json:"extraConfig"`
}

func (Site) TableName() string {
	return "nginx_sites"
}

type LocationRule struct {
	commonDomains.BaseDataEntity
	SiteGuid    string `gorm:"column:site_guid;size:50;index" json:"siteGuid"`
	Path        string `gorm:"column:path;size:255" json:"path"`
	ProxyPass   string `gorm:"column:proxy_pass;size:500" json:"proxyPass"`
	Root        string `gorm:"column:root;size:500" json:"root"`
	ExtraConfig string `gorm:"column:extra_config;type:text" json:"extraConfig"`
	Sort        int    `gorm:"column:sort" json:"sort"`
}

func (LocationRule) TableName() string {
	return "nginx_location_rules"
}
