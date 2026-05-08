package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

type Certificate struct {
	commonDomains.BaseDataEntity
	Name        string `gorm:"column:name;size:100;index" json:"name"`
	ServerName  string `gorm:"column:server_name;size:255;index" json:"serverName"`
	CertPath    string `gorm:"column:cert_path;size:500" json:"certPath"`
	KeyPath     string `gorm:"column:key_path;size:500" json:"keyPath"`
	Issuer      string `gorm:"column:issuer;size:255" json:"issuer"`
	NotBefore   int64  `gorm:"column:not_before" json:"notBefore"`
	NotAfter    int64  `gorm:"column:not_after;index" json:"notAfter"`
	AutoRenew   bool   `gorm:"column:auto_renew" json:"autoRenew"`
	Description string `gorm:"column:description;size:500" json:"description"`
}

func (Certificate) TableName() string {
	return "nginx_certificates"
}
