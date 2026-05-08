package inits

import (
	"nginx-go/domains"
	"nginx-go/routers"
	"nginx-go/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/wfu-work/nav-common-go-lib/global"
	commonInits "github.com/wfu-work/nav-common-go-lib/inits"
	"github.com/wfu-work/nav-common-go-lib/scheduleds"
	"go.uber.org/zap"
)

func Init() {
	sysInit := commonInits.SysInit{}
	sysInit.OnTableInit(registerTables)
	sysInit.OnRouterInit(func(publicGroup *gin.RouterGroup, privateGroup *gin.RouterGroup) {
		routers.RouterGroupApp.InitNginxRouter(privateGroup)
		routers.RouterGroupApp.InitInstanceRouter(privateGroup)
		routers.RouterGroupApp.InitSiteRouter(privateGroup)
		routers.RouterGroupApp.InitUpstreamRouter(privateGroup)
		routers.RouterGroupApp.InitCertificateRouter(privateGroup)
		routers.RouterGroupApp.InitConfigRouter(privateGroup)
		routers.RouterGroupApp.InitMetricRouter(privateGroup)
		routers.RouterGroupApp.InitLogRouter(privateGroup)
		routers.RouterGroupApp.InitSettingRouter(privateGroup)
		routers.RouterGroupApp.InitEventRouter(privateGroup)
	})
	sysInit.OnScheInit(registerSchedules)
	sysInit.Init()
}

func registerTables() {
	err := global.NAV_DB.AutoMigrate(
		domains.NginxOperation{},
		domains.NginxInstance{},
		domains.Site{},
		domains.LocationRule{},
		domains.Upstream{},
		domains.UpstreamServer{},
		domains.Certificate{},
		domains.ConfigVersion{},
		domains.PublishTask{},
		domains.MetricSample{},
		domains.Setting{},
		domains.AuditLog{},
	)
	if err != nil {
		global.NAV_LOG.Error("register nginx business table failed", zap.Error(err))
		return
	}
	global.NAV_LOG.Info("register nginx business table success")
}

func registerSchedules(timers scheduleds.Timer, options []cron.Option) {
	interval := metricCollectInterval()
	spec := "*/" + interval + " * * * * *"
	_, err := timers.AddTaskByFunc("NginxMetricCollect", spec, func() {
		if err := services.ServiceGroupApp.MetricService.Collect(); err != nil {
			global.NAV_LOG.Warn("collect nginx metrics failed", zap.Error(err))
		}
	}, "定时采集 Nginx 运行指标", options...)
	if err != nil {
		global.NAV_LOG.Error("register nginx metric collect task failed", zap.Error(err))
	}

	_, err = timers.AddTaskByFunc("NginxMetricRetention", "0 0 * * * *", func() {
		cutoff := time.Now().Add(-time.Duration(metricRetentionHours()) * time.Hour).UnixMilli()
		if err := global.NAV_DB.Where("create_time < ?", cutoff).Delete(&domains.MetricSample{}).Error; err != nil {
			global.NAV_LOG.Warn("clear nginx metric samples failed", zap.Error(err))
		}
	}, "每小时清理过期 Nginx 指标", options...)
	if err != nil {
		global.NAV_LOG.Error("register nginx metric retention task failed", zap.Error(err))
	}

	_, err = timers.AddTaskByFunc("NginxUpstreamHealthCollect", "*/30 * * * * *", func() {
		if err := services.ServiceGroupApp.UpstreamService.CollectHealth(); err != nil {
			global.NAV_LOG.Warn("collect nginx upstream health failed", zap.Error(err))
		}
	}, "定时检查 Nginx upstream 健康状态", options...)
	if err != nil {
		global.NAV_LOG.Error("register nginx upstream health task failed", zap.Error(err))
	}

	_, err = timers.AddTaskByFunc("NginxErrorLogScan", "0 */1 * * * *", func() {
		if err := services.ServiceGroupApp.LogService.ScanErrors(); err != nil {
			global.NAV_LOG.Warn("scan nginx error log failed", zap.Error(err))
		}
	}, "定时扫描 Nginx error log 关键错误", options...)
	if err != nil {
		global.NAV_LOG.Error("register nginx error log scan task failed", zap.Error(err))
	}
}

func metricCollectInterval() string {
	seconds := 30
	if global.NAV_VIPER != nil {
		seconds = global.NAV_VIPER.GetInt("metrics.collect-interval-seconds")
	}
	if seconds <= 0 {
		seconds = 30
	}
	if seconds > 59 {
		seconds = 59
	}
	return strconv.Itoa(seconds)
}

func metricRetentionHours() int {
	hours := 72
	if global.NAV_VIPER != nil {
		hours = global.NAV_VIPER.GetInt("metrics.retention-hours")
	}
	if hours <= 0 {
		return 72
	}
	return hours
}
