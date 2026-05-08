package services

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	NginxService
	InstanceService
	SiteService
	UpstreamService
	CertificateService
	ConfigService
	MetricService
	LogService
	SettingService
	AuditService
}
