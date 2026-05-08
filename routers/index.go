package routers

import "nginx-go/apis"

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	NginxRouter
	InstanceRouter
	SiteRouter
	UpstreamRouter
	CertificateRouter
	ConfigRouter
	MetricRouter
	LogRouter
	SettingRouter
	EventRouter
}

var (
	nginxApi    = apis.ApiGroupApp.NginxApi
	instanceApi = apis.ApiGroupApp.InstanceApi
	siteApi     = apis.ApiGroupApp.SiteApi
	upstreamApi = apis.ApiGroupApp.UpstreamApi
	certApi     = apis.ApiGroupApp.CertificateApi
	configApi   = apis.ApiGroupApp.ConfigApi
	metricApi   = apis.ApiGroupApp.MetricApi
	logApi      = apis.ApiGroupApp.LogApi
	settingApi  = apis.ApiGroupApp.SettingApi
	eventApi    = apis.ApiGroupApp.EventApi
)
