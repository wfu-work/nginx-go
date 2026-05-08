package apis

import "nginx-go/services"

var ApiGroupApp = new(ApiGroup)

type ApiGroup struct {
	NginxApi
	InstanceApi
	SiteApi
	UpstreamApi
	CertificateApi
	ConfigApi
	MetricApi
	LogApi
	SettingApi
	EventApi
}

var (
	nginxService    = services.ServiceGroupApp.NginxService
	instanceService = services.ServiceGroupApp.InstanceService
	siteService     = services.ServiceGroupApp.SiteService
	upstreamService = services.ServiceGroupApp.UpstreamService
	certService     = services.ServiceGroupApp.CertificateService
	configService   = services.ServiceGroupApp.ConfigService
	metricService   = services.ServiceGroupApp.MetricService
	logService      = services.ServiceGroupApp.LogService
	settingService  = services.ServiceGroupApp.SettingService
	auditService    = services.ServiceGroupApp.AuditService
)
