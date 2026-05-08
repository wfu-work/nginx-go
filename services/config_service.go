package services

import (
	"bytes"
	"errors"
	"fmt"
	"nginx-go/domains"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
	"gorm.io/gorm"
)

type ConfigService struct {
	commonServices.CrudService[domains.ConfigVersion]
	taskCrud commonServices.CrudService[domains.PublishTask]
}

type RenderRequest struct {
	SiteGuid string `json:"siteGuid"`
	Save     bool   `json:"save"`
	Reason   string `json:"reason"`
}

type ValidateRequest struct {
	InstanceGuid string `json:"instanceGuid"`
	SiteGuid     string `json:"siteGuid"`
	Config       string `json:"config"`
	Save         bool   `json:"save"`
	Reason       string `json:"reason"`
}

type PublishRequest struct {
	InstanceGuid string `json:"instanceGuid"`
	VersionGuid  string `json:"versionGuid"`
	SiteGuid     string `json:"siteGuid"`
	Config       string `json:"config"`
	Reason       string `json:"reason"`
}

type RollbackRequest struct {
	InstanceGuid string `json:"instanceGuid"`
	VersionGuid  string `json:"versionGuid"`
	Confirm      bool   `json:"confirm"`
	Reason       string `json:"reason"`
}

type DiffRequest struct {
	FromVersionGuid string `json:"fromVersionGuid"`
	ToVersionGuid   string `json:"toVersionGuid"`
	FromConfig      string `json:"fromConfig"`
	ToConfig        string `json:"toConfig"`
}

type RenderResult struct {
	Config      string `json:"config"`
	VersionGuid string `json:"versionGuid,omitempty"`
	VersionNo   int64  `json:"versionNo,omitempty"`
}

type ValidateResult struct {
	VersionGuid string `json:"versionGuid,omitempty"`
	VersionNo   int64  `json:"versionNo,omitempty"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Output      string `json:"output"`
	ConfigPath  string `json:"configPath"`
	DurationMs  int64  `json:"durationMs"`
}

type PublishResult struct {
	TaskGuid      string `json:"taskGuid"`
	VersionGuid   string `json:"versionGuid"`
	OperationGuid string `json:"operationGuid"`
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	TargetPath    string `json:"targetPath"`
	BackupPath    string `json:"backupPath"`
	DurationMs    int64  `json:"durationMs"`
}

type DiffResult struct {
	DiffText string `json:"diffText"`
	HTML     string `json:"html"`
}

func (s ConfigService) Render(req RenderRequest) (RenderResult, error) {
	config, err := s.renderConfig(req.SiteGuid)
	if err != nil {
		return RenderResult{}, err
	}
	result := RenderResult{Config: config}
	if req.Save {
		version, err := s.createVersion(req.SiteGuid, config, domains.ConfigVersionStatusRendered, false, "", req.Reason, "")
		if err != nil {
			return RenderResult{}, err
		}
		result.VersionGuid = version.Guid
		result.VersionNo = version.VersionNo
	}
	return result, nil
}

func (s ConfigService) Validate(req ValidateRequest) (ValidateResult, error) {
	config := req.Config
	if config == "" {
		rendered, err := s.renderConfig(req.SiteGuid)
		if err != nil {
			return ValidateResult{}, err
		}
		config = rendered
	}
	configPath, err := writeTempConfig(config)
	if err != nil {
		return ValidateResult{}, err
	}
	nginxResult, _ := ServiceGroupApp.NginxService.Test(OperationRequest{InstanceGuid: req.InstanceGuid, ConfigPath: configPath, Reason: req.Reason})
	result := ValidateResult{
		Success:    nginxResult.Success,
		Message:    nginxResult.Message,
		Output:     nginxResult.Output,
		ConfigPath: configPath,
		DurationMs: nginxResult.DurationMs,
	}
	if req.Save {
		status := domains.ConfigVersionStatusValidated
		version, err := s.createVersion(req.SiteGuid, config, status, nginxResult.Success, nginxResult.Output, req.Reason, "")
		if err != nil {
			return ValidateResult{}, err
		}
		result.VersionGuid = version.Guid
		result.VersionNo = version.VersionNo
	}
	if !nginxResult.Success {
		return result, errors.New(nginxResult.Message)
	}
	return result, nil
}

func (s ConfigService) Publish(req PublishRequest) (PublishResult, error) {
	version, err := s.resolvePublishVersion(req)
	if err != nil {
		return PublishResult{}, err
	}
	return s.publishVersion(version, req.InstanceGuid, "publish", req.Reason, "")
}

func (s ConfigService) Rollback(req RollbackRequest) (PublishResult, error) {
	if req.VersionGuid == "" {
		return PublishResult{}, errors.New("missing version guid")
	}
	if !req.Confirm && containsString(confirmActions(), "rollback") {
		return PublishResult{}, errors.New("rollback requires confirm=true")
	}
	version, err := s.GetByGuid(req.VersionGuid)
	if err != nil {
		return PublishResult{}, err
	}
	if version == nil {
		return PublishResult{}, errors.New("config version not found")
	}
	rollbackVersion, err := s.createVersion(version.SiteGuid, version.Config, domains.ConfigVersionStatusRolledBack, true, version.ValidateMsg, req.Reason, version.Guid)
	if err != nil {
		return PublishResult{}, err
	}
	return s.publishVersion(rollbackVersion, req.InstanceGuid, "rollback", req.Reason, version.Guid)
}

func (s ConfigService) Diff(req DiffRequest) (DiffResult, error) {
	fromConfig := req.FromConfig
	toConfig := req.ToConfig
	if req.FromVersionGuid != "" {
		version, err := s.VersionGet(req.FromVersionGuid)
		if err != nil {
			return DiffResult{}, err
		}
		fromConfig = version.Config
	}
	if req.ToVersionGuid != "" {
		version, err := s.VersionGet(req.ToVersionGuid)
		if err != nil {
			return DiffResult{}, err
		}
		toConfig = version.Config
	}
	if fromConfig == "" && toConfig == "" {
		return DiffResult{}, errors.New("missing diff config")
	}
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(fromConfig, toConfig, false)
	return DiffResult{
		DiffText: dmp.DiffPrettyText(diffs),
		HTML:     dmp.DiffPrettyHtml(diffs),
	}, nil
}

func (s ConfigService) VersionList(params map[string]string) (interface{}, int64, error) {
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Desc == "" && pageInfo.Asc == "" {
		pageInfo.Desc = "createTime"
	}
	return s.List(pageInfo, "status,reason")
}

func (s ConfigService) VersionGet(guid string) (*domains.ConfigVersion, error) {
	if guid == "" {
		return nil, errors.New("missing version guid")
	}
	version, err := s.GetByGuid(guid)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, errors.New("config version not found")
	}
	return version, nil
}

func (s ConfigService) TaskList(params map[string]string) (interface{}, int64, error) {
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Desc == "" && pageInfo.Asc == "" {
		pageInfo.Desc = "createTime"
	}
	return s.taskCrud.List(pageInfo, "action,status,message,reason")
}

func (s ConfigService) renderConfig(siteGuid string) (string, error) {
	var sites []domains.Site
	query := global.NAV_DB.Where("enabled = ?", true)
	if siteGuid != "" {
		query = query.Where("guid = ?", siteGuid)
	}
	if err := query.Order("id asc").Find(&sites).Error; err != nil {
		return "", err
	}
	if siteGuid != "" && len(sites) == 0 {
		return "", errors.New("site not found")
	}
	var upstreams []domains.Upstream
	if err := global.NAV_DB.Order("id asc").Find(&upstreams).Error; err != nil {
		return "", err
	}
	data := struct {
		Upstreams []renderUpstream
		Sites     []renderSite
	}{
		Upstreams: renderUpstreams(upstreams),
		Sites:     renderSites(sites),
	}
	tpl, err := template.New("nginx").Parse(nginxTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s ConfigService) resolvePublishVersion(req PublishRequest) (*domains.ConfigVersion, error) {
	if req.VersionGuid != "" {
		return s.VersionGet(req.VersionGuid)
	}
	config := req.Config
	if config == "" {
		rendered, err := s.renderConfig(req.SiteGuid)
		if err != nil {
			return nil, err
		}
		config = rendered
	}
	return s.createVersion(req.SiteGuid, config, domains.ConfigVersionStatusValidated, true, "", req.Reason, "")
}

func (s ConfigService) createVersion(siteGuid, config, status string, validateOK bool, validateMsg, reason, rollbackFrom string) (*domains.ConfigVersion, error) {
	versionNo, err := nextConfigVersionNo()
	if err != nil {
		return nil, err
	}
	version := domains.ConfigVersion{
		BaseDataEntity: commonDomains.BaseDataEntity{},
		SiteGuid:       siteGuid,
		VersionNo:      versionNo,
		Status:         status,
		Config:         config,
		ValidateOK:     validateOK,
		ValidateMsg:    validateMsg,
		Reason:         reason,
		RollbackFrom:   rollbackFrom,
	}
	if err := global.NAV_DB.Create(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (s ConfigService) recordPublishTask(versionGuid, action string, success bool, targetPath, backupPath, message, operationGuid string, durationMs int64, reason string) (*domains.PublishTask, error) {
	task := domains.PublishTask{
		BaseDataEntity: commonDomains.BaseDataEntity{Guid: strings.ReplaceAll(uuid.NewString(), "-", "")},
		VersionGuid:    versionGuid,
		Action:         action,
		Success:        success,
		Status:         statusText(success),
		TargetPath:     targetPath,
		BackupPath:     backupPath,
		Message:        message,
		OperationGuid:  operationGuid,
		DurationMs:     durationMs,
		Reason:         reason,
	}
	if err := s.taskCrud.Create(task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (s ConfigService) publishVersion(version *domains.ConfigVersion, instanceGuid, action, reason, rollbackFrom string) (PublishResult, error) {
	start := time.Now()
	runtime, err := resolveNginxRuntime(instanceGuid)
	if err != nil {
		return PublishResult{}, err
	}
	validateResult, err := s.Validate(ValidateRequest{InstanceGuid: runtime.InstanceGuid, Config: version.Config, SiteGuid: version.SiteGuid, Reason: reason})
	if err != nil {
		task, taskErr := s.recordPublishTask(version.Guid, action, false, "", "", validateResult.Message, "", time.Since(start).Milliseconds(), reason)
		if taskErr != nil {
			return PublishResult{}, taskErr
		}
		result := publishResultFromTask(task, version.Guid)
		s.auditConfigChange(action, runtime.InstanceGuid, result, reason, rollbackFrom)
		return result, err
	}
	targetPath := managedConfigPath(runtime)
	backupPath, err := backupExistingConfig(targetPath)
	if err != nil {
		return PublishResult{}, err
	}
	if err := atomicWriteFile(targetPath, []byte(version.Config), 0644); err != nil {
		return PublishResult{}, err
	}
	operation, reloadErr := ServiceGroupApp.NginxService.Reload(OperationRequest{InstanceGuid: runtime.InstanceGuid, ConfigPath: targetPath, Confirm: true, Reason: reason})
	success := reloadErr == nil && operation.Success
	message := operation.Message
	if reloadErr != nil {
		message = reloadErr.Error()
	}
	task, err := s.recordPublishTask(version.Guid, action, success, targetPath, backupPath, message, operation.OperationGuid, time.Since(start).Milliseconds(), reason)
	if err != nil {
		return PublishResult{}, err
	}
	if success {
		now := time.Now().UnixMilli()
		status := domains.ConfigVersionStatusPublished
		if action == "rollback" {
			status = domains.ConfigVersionStatusRolledBack
		}
		global.NAV_DB.Model(&domains.ConfigVersion{}).Where("guid = ?", version.Guid).Updates(map[string]any{
			"status":       status,
			"validate_ok":  true,
			"validate_msg": validateResult.Output,
			"published_at": now,
		})
	} else if backupPath != "" {
		_ = restoreBackup(targetPath, backupPath)
	}
	result := publishResultFromTask(task, version.Guid)
	s.auditConfigChange(action, runtime.InstanceGuid, result, reason, rollbackFrom)
	return result, reloadErr
}

func (s ConfigService) auditConfigChange(action, instanceGuid string, result PublishResult, reason, rollbackFrom string) {
	auditAction := domains.AuditActionConfigPublish
	if action == "rollback" {
		auditAction = domains.AuditActionConfigRollback
	}
	ServiceGroupApp.AuditService.Record(AuditRecord{
		Action:       auditAction,
		ResourceType: "config_version",
		ResourceGuid: result.VersionGuid,
		Success:      result.Success,
		Message:      result.Message,
		Reason:       reason,
		Detail: map[string]any{
			"taskGuid":      result.TaskGuid,
			"operationGuid": result.OperationGuid,
			"instanceGuid":  instanceGuid,
			"targetPath":    result.TargetPath,
			"backupPath":    result.BackupPath,
			"rollbackFrom":  rollbackFrom,
		},
	})
}

func publishResultFromTask(task *domains.PublishTask, versionGuid string) PublishResult {
	return PublishResult{
		TaskGuid:      task.Guid,
		VersionGuid:   versionGuid,
		OperationGuid: task.OperationGuid,
		Success:       task.Success,
		Message:       task.Message,
		TargetPath:    task.TargetPath,
		BackupPath:    task.BackupPath,
		DurationMs:    task.DurationMs,
	}
}

func nextConfigVersionNo() (int64, error) {
	var latest domains.ConfigVersion
	err := global.NAV_DB.Order("version_no desc").First(&latest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Now().UnixMilli(), nil
	}
	if err != nil {
		return 0, err
	}
	if latest.VersionNo == 0 {
		return time.Now().UnixMilli(), nil
	}
	return latest.VersionNo + 1, nil
}

func writeTempConfig(config string) (string, error) {
	dir := tempConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("nginx-%d.conf", time.Now().UnixNano()))
	return path, os.WriteFile(path, []byte(config), 0644)
}

func backupExistingConfig(targetPath string) (string, error) {
	if _, err := os.Stat(targetPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	backupDir := backupConfigDir()
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s.%d.bak", filepath.Base(targetPath), time.Now().UnixMilli()))
	content, err := os.ReadFile(targetPath)
	if err != nil {
		return "", err
	}
	return backupPath, os.WriteFile(backupPath, content, 0644)
}

func restoreBackup(targetPath, backupPath string) error {
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	return atomicWriteFile(targetPath, content, 0644)
}

func atomicWriteFile(path string, content []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, content, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func managedConfigPath(runtime nginxRuntime) string {
	path := runtime.ManagedConfig
	if path == "" {
		return "./data/nginx/generated.conf"
	}
	return path
}

func tempConfigDir() string {
	return configString("nginx.temp-dir", "./data/nginx/tmp")
}

func backupConfigDir() string {
	return configString("nginx.backup-dir", "./data/nginx/backups")
}

type renderSite struct {
	domains.Site
	Locations []domains.LocationRule
	Cert      *domains.Certificate
}

type renderUpstream struct {
	domains.Upstream
	Servers []domains.UpstreamServer
}

func renderSites(sites []domains.Site) []renderSite {
	result := make([]renderSite, 0, len(sites))
	for _, site := range sites {
		var locations []domains.LocationRule
		global.NAV_DB.Where("site_guid = ?", site.Guid).Order("sort asc, id asc").Find(&locations)
		var cert *domains.Certificate
		if site.CertificateGuid != "" {
			var found domains.Certificate
			dbResult := global.NAV_DB.Where("guid = ?", site.CertificateGuid).Find(&found)
			if dbResult.Error == nil && dbResult.RowsAffected > 0 {
				cert = &found
			}
		}
		result = append(result, renderSite{Site: site, Locations: locations, Cert: cert})
	}
	return result
}

func renderUpstreams(upstreams []domains.Upstream) []renderUpstream {
	result := make([]renderUpstream, 0, len(upstreams))
	for _, upstream := range upstreams {
		var servers []domains.UpstreamServer
		global.NAV_DB.Where("upstream_guid = ?", upstream.Guid).Order("sort asc, id asc").Find(&servers)
		result = append(result, renderUpstream{Upstream: upstream, Servers: servers})
	}
	return result
}

const nginxTemplate = `{{- range .Upstreams }}
upstream {{ .Name }} {
{{- if .Method }}
    {{ .Method }};
{{- end }}
{{- range .Servers }}
    server {{ .Address }} weight={{ .Weight }} max_fails={{ .MaxFails }} fail_timeout={{ .FailTimeout }}{{ if .Backup }} backup{{ end }}{{ if .Down }} down{{ end }};
{{- end }}
{{- if .ExtraConfig }}
{{ .ExtraConfig }}
{{- end }}
}

{{ end -}}
{{ range .Sites }}
server {
    listen {{ .Listen }};
{{- if .SSL }}
    listen 443 ssl;
{{- if .Cert }}
    ssl_certificate {{ .Cert.CertPath }};
    ssl_certificate_key {{ .Cert.KeyPath }};
{{- end }}
{{- end }}
    server_name {{ .ServerName }};
{{- if .Root }}
    root {{ .Root }};
{{- end }}
{{- if .Index }}
    index {{ .Index }};
{{- end }}
{{- if .AccessLog }}
    access_log {{ .AccessLog }};
{{- end }}
{{- if .ErrorLog }}
    error_log {{ .ErrorLog }};
{{- end }}
{{- range .Locations }}

    location {{ .Path }} {
{{- if .ProxyPass }}
        proxy_pass {{ .ProxyPass }};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
{{- end }}
{{- if .Root }}
        root {{ .Root }};
{{- end }}
{{- if .ExtraConfig }}
{{ .ExtraConfig }}
{{- end }}
    }
{{- end }}
{{- if .ExtraConfig }}
{{ .ExtraConfig }}
{{- end }}
}

{{ end -}}`
