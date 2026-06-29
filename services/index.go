package services

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	NginxService
	NodeService
	AgentService
	InstanceService
	SiteService
	UpstreamService
	CertificateService
	ConfigService
	MetricService
	LogService
	SettingService
	AuditService
	EventNotificationService
}
