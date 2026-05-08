package services

import (
	"errors"
	"fmt"
	"nginx-go/domains"
	commandUtils "nginx-go/utils"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/process"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type NginxService struct {
	commonServices.CrudService[domains.NginxOperation]
}

type OperationRequest struct {
	InstanceGuid string `json:"instanceGuid"`
	ConfigPath   string `json:"configPath"`
	Confirm      bool   `json:"confirm"`
	Reason       string `json:"reason"`
}

type OperationResult struct {
	OperationGuid string `json:"operationGuid"`
	Action        string `json:"action"`
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	Command       string `json:"command"`
	Output        string `json:"output"`
	DurationMs    int64  `json:"durationMs"`
}

type StatusResult struct {
	InstanceGuid string          `json:"instanceGuid"`
	Mode         string          `json:"mode"`
	Running      bool            `json:"running"`
	Message      string          `json:"message"`
	Version      string          `json:"version"`
	Processes    []ProcessStatus `json:"processes"`
	CheckedAt    int64           `json:"checkedAt"`
}

type nginxRuntime struct {
	InstanceGuid    string
	Mode            string
	Bin             string
	Systemctl       string
	ServiceName     string
	MainConfig      string
	ManagedConfig   string
	DockerBin       string
	DockerContainer string
	AccessLog       string
	ErrorLog        string
	StubStatusURL   string
	Timeout         time.Duration
}

type ProcessStatus struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpuPercent"`
	MemoryMB   float32 `json:"memoryMb"`
}

func (s NginxService) Status(instanceGuid string) (StatusResult, error) {
	runtime, err := resolveNginxRuntime(instanceGuid)
	if err != nil {
		return StatusResult{}, err
	}
	processes := nginxProcesses()
	version := nginxVersion(runtime)
	result := StatusResult{
		InstanceGuid: runtime.InstanceGuid,
		Mode:         runtime.Mode,
		Running:      len(processes) > 0 || systemdIsActive(runtime) || dockerIsRunning(runtime),
		Message:      "nginx status refreshed",
		Version:      version.Output,
		Processes:    processes,
		CheckedAt:    time.Now().UnixMilli(),
	}
	if !result.Running {
		result.Message = "nginx is not running or cannot be detected"
	}
	return result, nil
}

func (s NginxService) Refresh(req OperationRequest) (OperationResult, error) {
	return s.recordOnly(domains.NginxActionRefresh, req, true, "nginx status refreshed", "", "", 0)
}

func (s NginxService) Test(req OperationRequest) (OperationResult, error) {
	runtime, err := resolveNginxRuntime(req.InstanceGuid)
	if err != nil {
		return OperationResult{}, err
	}
	req.InstanceGuid = runtime.InstanceGuid
	args := []string{"-t"}
	if req.ConfigPath != "" {
		args = append(args, "-c", req.ConfigPath)
	} else if runtime.MainConfig != "" {
		args = append(args, "-c", runtime.MainConfig)
	}
	result := runNginx(runtime, args...)
	return s.recordCommand(domains.NginxActionTest, req, result, "nginx config test success", "nginx config test failed")
}

func (s NginxService) Reload(req OperationRequest) (OperationResult, error) {
	testResult, err := s.Test(req)
	if err != nil || !testResult.Success {
		if err != nil {
			return testResult, err
		}
		return testResult, errors.New(testResult.Message)
	}
	return s.runServiceAction(domains.NginxActionReload, req)
}

func (s NginxService) Restart(req OperationRequest) (OperationResult, error) {
	if err := requireConfirm(domains.NginxActionRestart, req); err != nil {
		return OperationResult{}, err
	}
	return s.runServiceAction(domains.NginxActionRestart, req)
}

func (s NginxService) Start(req OperationRequest) (OperationResult, error) {
	return s.runServiceAction(domains.NginxActionStart, req)
}

func (s NginxService) Stop(req OperationRequest) (OperationResult, error) {
	if err := requireConfirm(domains.NginxActionStop, req); err != nil {
		return OperationResult{}, err
	}
	return s.runServiceAction(domains.NginxActionStop, req)
}

func (s NginxService) OperationList(params map[string]string) (interface{}, int64, error) {
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Desc == "" && pageInfo.Asc == "" {
		pageInfo.Desc = "createTime"
	}
	return s.List(pageInfo, "action,status,message,reason")
}

func (s NginxService) OperationGet(guid string) (*domains.NginxOperation, error) {
	if guid == "" {
		return nil, errors.New("missing operation guid")
	}
	result, err := s.GetByGuid(guid)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("operation not found")
	}
	return result, nil
}

func (s NginxService) runServiceAction(action string, req OperationRequest) (OperationResult, error) {
	runtime, err := resolveNginxRuntime(req.InstanceGuid)
	if err != nil {
		return OperationResult{}, err
	}
	req.InstanceGuid = runtime.InstanceGuid
	var result commandUtils.CommandResult
	switch runtime.Mode {
	case "systemd":
		result = commandUtils.RunCommand(runtime.Timeout, runtime.Systemctl, action, runtime.ServiceName)
	case "command":
		result = runCommandMode(runtime, action)
	case "docker":
		result = runDockerMode(runtime, action)
	default:
		return OperationResult{}, fmt.Errorf("unsupported nginx mode: %s", runtime.Mode)
	}
	return s.recordCommand(action, req, result, fmt.Sprintf("nginx %s success", action), fmt.Sprintf("nginx %s failed", action))
}

func runCommandMode(runtime nginxRuntime, action string) commandUtils.CommandResult {
	switch action {
	case domains.NginxActionReload:
		return commandUtils.RunCommand(runtime.Timeout, runtime.Bin, "-s", "reload")
	case domains.NginxActionStop:
		return commandUtils.RunCommand(runtime.Timeout, runtime.Bin, "-s", "stop")
	case domains.NginxActionStart:
		return commandUtils.RunCommand(runtime.Timeout, runtime.Bin)
	case domains.NginxActionRestart:
		stop := commandUtils.RunCommand(runtime.Timeout, runtime.Bin, "-s", "stop")
		if !stop.Success {
			return stop
		}
		return commandUtils.RunCommand(runtime.Timeout, runtime.Bin)
	default:
		return commandUtils.CommandResult{Success: false, Output: "unsupported command action"}
	}
}

func runDockerMode(runtime nginxRuntime, action string) commandUtils.CommandResult {
	if runtime.DockerContainer == "" {
		return commandUtils.CommandResult{Success: false, Output: "nginx docker container is empty"}
	}
	switch action {
	case domains.NginxActionReload:
		return commandUtils.RunCommand(runtime.Timeout, runtime.DockerBin, "exec", runtime.DockerContainer, runtime.Bin, "-s", "reload")
	case domains.NginxActionStop:
		return commandUtils.RunCommand(runtime.Timeout, runtime.DockerBin, "stop", runtime.DockerContainer)
	case domains.NginxActionStart:
		return commandUtils.RunCommand(runtime.Timeout, runtime.DockerBin, "start", runtime.DockerContainer)
	case domains.NginxActionRestart:
		return commandUtils.RunCommand(runtime.Timeout, runtime.DockerBin, "restart", runtime.DockerContainer)
	default:
		return commandUtils.CommandResult{Success: false, Output: "unsupported docker action"}
	}
}

func (s NginxService) recordCommand(action string, req OperationRequest, cmd commandUtils.CommandResult, successMsg, failMsg string) (OperationResult, error) {
	message := successMsg
	if !cmd.Success {
		message = failMsg
	}
	return s.recordOnly(action, req, cmd.Success, message, cmd.Command, cmd.Output, cmd.DurationMs)
}

func (s NginxService) recordOnly(action string, req OperationRequest, success bool, message, command, output string, durationMs int64) (OperationResult, error) {
	if req.InstanceGuid == "" {
		req.InstanceGuid = "default"
	}
	op := domains.NginxOperation{
		BaseDataEntity: commonDomains.BaseDataEntity{Guid: strings.ReplaceAll(uuid.NewString(), "-", "")},
		InstanceGuid:   req.InstanceGuid,
		Action:         action,
		Success:        success,
		Status:         statusText(success),
		Message:        message,
		Command:        command,
		Output:         output,
		DurationMs:     durationMs,
		Reason:         req.Reason,
	}
	if global.NAV_DB != nil {
		if err := s.Create(op); err != nil {
			global.NAV_LOG.Error("record nginx operation failed", zap.Error(err))
			return OperationResult{}, err
		}
	}
	if isAuditedNginxAction(action) {
		ServiceGroupApp.AuditService.Record(AuditRecord{
			Action:       domains.AuditActionNginxOperation,
			ResourceType: "nginx_operation",
			ResourceGuid: op.Guid,
			Success:      success,
			Message:      message,
			Reason:       req.Reason,
			Detail: map[string]any{
				"instanceGuid": req.InstanceGuid,
				"action":       action,
				"command":      command,
				"durationMs":   durationMs,
			},
		})
	}
	return OperationResult{
		OperationGuid: op.Guid,
		Action:        action,
		Success:       success,
		Message:       message,
		Command:       command,
		Output:        output,
		DurationMs:    durationMs,
	}, nil
}

func isAuditedNginxAction(action string) bool {
	switch action {
	case domains.NginxActionReload, domains.NginxActionRestart, domains.NginxActionStart, domains.NginxActionStop, domains.NginxActionTest:
		return true
	default:
		return false
	}
}

func requireConfirm(action string, req OperationRequest) error {
	if !req.Confirm && containsString(confirmActions(), action) {
		return fmt.Errorf("%s requires confirm=true", action)
	}
	return nil
}

func nginxProcesses() []ProcessStatus {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}
	results := make([]ProcessStatus, 0)
	for _, item := range processes {
		name, err := item.Name()
		if err != nil || !strings.Contains(strings.ToLower(name), "nginx") {
			continue
		}
		cpuPercent, _ := item.CPUPercent()
		mem, _ := item.MemoryInfo()
		memoryMB := float32(0)
		if mem != nil {
			memoryMB = float32(mem.RSS) / 1024 / 1024
		}
		results = append(results, ProcessStatus{
			PID:        item.Pid,
			Name:       name,
			CPUPercent: cpuPercent,
			MemoryMB:   memoryMB,
		})
	}
	return results
}

func systemdIsActive(runtime nginxRuntime) bool {
	if runtime.Mode != "systemd" {
		return false
	}
	result := commandUtils.RunCommand(runtime.Timeout, runtime.Systemctl, "is-active", runtime.ServiceName)
	return result.Success && strings.TrimSpace(result.Output) == "active"
}

func dockerIsRunning(runtime nginxRuntime) bool {
	if runtime.Mode != "docker" || runtime.DockerContainer == "" {
		return false
	}
	result := commandUtils.RunCommand(runtime.Timeout, runtime.DockerBin, "inspect", "-f", "{{.State.Running}}", runtime.DockerContainer)
	return result.Success && strings.TrimSpace(result.Output) == "true"
}

func nginxVersion(runtime nginxRuntime) commandUtils.CommandResult {
	return runNginx(runtime, "-v")
}

func runNginx(runtime nginxRuntime, args ...string) commandUtils.CommandResult {
	if runtime.Mode == "docker" {
		if runtime.DockerContainer == "" {
			return commandUtils.CommandResult{Success: false, Output: "nginx docker container is empty"}
		}
		dockerArgs := append([]string{"exec", runtime.DockerContainer, runtime.Bin}, args...)
		return commandUtils.RunCommand(runtime.Timeout, runtime.DockerBin, dockerArgs...)
	}
	return commandUtils.RunCommand(runtime.Timeout, runtime.Bin, args...)
}

func statusText(success bool) string {
	if success {
		return "success"
	}
	return "failed"
}

func nginxMode() string {
	return configString("nginx.mode", "command")
}

func nginxBin() string {
	return configString("nginx.bin", "nginx")
}

func systemctlBin() string {
	return configString("nginx.systemctl", "systemctl")
}

func serviceName() string {
	return configString("nginx.service-name", "nginx")
}

func mainConfig() string {
	return configString("nginx.main-config", "")
}

func resolveNginxRuntime(instanceGuid string) (nginxRuntime, error) {
	runtime := nginxRuntime{
		InstanceGuid:    defaultInstanceGuid(instanceGuid),
		Mode:            nginxMode(),
		Bin:             nginxBin(),
		Systemctl:       systemctlBin(),
		ServiceName:     serviceName(),
		MainConfig:      mainConfig(),
		ManagedConfig:   configString("nginx.managed-config", "./data/nginx/generated.conf"),
		DockerBin:       configString("nginx.docker-bin", "docker"),
		DockerContainer: configString("nginx.docker-container", ""),
		AccessLog:       configString("nginx.access-log", "/var/log/nginx/access.log"),
		ErrorLog:        configString("nginx.error-log", "/var/log/nginx/error.log"),
		StubStatusURL:   configString("nginx.stub-status-url", ""),
		Timeout:         timeout(),
	}
	if global.NAV_DB == nil {
		return runtime, nil
	}
	var instance domains.NginxInstance
	result := global.NAV_DB.Where("guid = ?", runtime.InstanceGuid).Find(&instance)
	if result.Error != nil {
		return runtime, result.Error
	}
	if result.RowsAffected == 0 {
		if runtime.InstanceGuid == "default" {
			return runtime, nil
		}
		return runtime, fmt.Errorf("nginx instance not found: %s", runtime.InstanceGuid)
	}
	if !instance.Enabled {
		return runtime, fmt.Errorf("nginx instance is disabled: %s", runtime.InstanceGuid)
	}
	applyInstanceRuntime(&runtime, instance)
	return runtime, nil
}

func listNginxRuntimes() ([]nginxRuntime, error) {
	if global.NAV_DB == nil {
		runtime, err := resolveNginxRuntime("")
		if err != nil {
			return nil, err
		}
		return []nginxRuntime{runtime}, nil
	}
	var instances []domains.NginxInstance
	if err := global.NAV_DB.Where("enabled = ?", true).Order("id asc").Find(&instances).Error; err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		runtime, err := resolveNginxRuntime("")
		if err != nil {
			return nil, err
		}
		return []nginxRuntime{runtime}, nil
	}
	runtimes := make([]nginxRuntime, 0, len(instances))
	for _, instance := range instances {
		runtime, err := resolveNginxRuntime(instance.Guid)
		if err != nil {
			return nil, err
		}
		runtimes = append(runtimes, runtime)
	}
	return runtimes, nil
}

func applyInstanceRuntime(runtime *nginxRuntime, instance domains.NginxInstance) {
	if instance.Mode != "" {
		runtime.Mode = instance.Mode
	}
	if instance.Bin != "" {
		runtime.Bin = instance.Bin
	}
	if instance.Systemctl != "" {
		runtime.Systemctl = instance.Systemctl
	}
	if instance.ServiceName != "" {
		runtime.ServiceName = instance.ServiceName
	}
	if instance.MainConfig != "" {
		runtime.MainConfig = instance.MainConfig
	}
	if instance.ManagedConfig != "" {
		runtime.ManagedConfig = instance.ManagedConfig
	}
	if instance.DockerContainer != "" {
		runtime.DockerContainer = instance.DockerContainer
	}
	if instance.AccessLog != "" {
		runtime.AccessLog = instance.AccessLog
	}
	if instance.ErrorLog != "" {
		runtime.ErrorLog = instance.ErrorLog
	}
	if instance.StubStatusURL != "" {
		runtime.StubStatusURL = instance.StubStatusURL
	}
}

func defaultInstanceGuid(instanceGuid string) string {
	if instanceGuid != "" {
		return instanceGuid
	}
	return configString("nginx.default-instance", "default")
}

func timeout() time.Duration {
	seconds := 10
	if global.NAV_VIPER != nil {
		seconds = global.NAV_VIPER.GetInt("nginx.command-timeout-seconds")
	}
	if seconds <= 0 {
		seconds = 10
	}
	return time.Duration(seconds) * time.Second
}

func confirmActions() []string {
	if global.NAV_VIPER == nil {
		return []string{domains.NginxActionRestart, domains.NginxActionStop, "rollback"}
	}
	actions := global.NAV_VIPER.GetStringSlice("nginx.require-confirm-actions")
	if len(actions) == 0 {
		return []string{domains.NginxActionRestart, domains.NginxActionStop, "rollback"}
	}
	return actions
}

func configString(key, fallback string) string {
	if global.NAV_DB != nil {
		var setting domains.Setting
		result := global.NAV_DB.Where("key = ?", key).Find(&setting)
		if result.Error == nil && result.RowsAffected > 0 && setting.Value != "" {
			return setting.Value
		}
	}
	if global.NAV_VIPER == nil {
		return fallback
	}
	value := global.NAV_VIPER.GetString(key)
	if value == "" {
		return fallback
	}
	return value
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func executableExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
