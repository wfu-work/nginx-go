package services

import "time"

const (
	AgentTaskNginxStatus         = "nginx.status"
	AgentTaskNginxOperation      = "nginx.operation"
	AgentTaskNginxStubStatus     = "nginx.stub_status"
	AgentTaskNginxProcess        = "nginx.process"
	AgentTaskNginxLogTail        = "nginx.log_tail"
	AgentTaskNginxConfigValidate = "nginx.config_validate"
	AgentTaskNginxConfigPublish  = "nginx.config_publish"
)

type AgentRuntimeSpec struct {
	InstanceGuid    string `json:"instanceGuid"`
	Mode            string `json:"mode"`
	Bin             string `json:"bin"`
	Systemctl       string `json:"systemctl"`
	ServiceName     string `json:"serviceName"`
	MainConfig      string `json:"mainConfig"`
	ManagedConfig   string `json:"managedConfig"`
	DockerBin       string `json:"dockerBin"`
	DockerContainer string `json:"dockerContainer"`
	AccessLog       string `json:"accessLog"`
	ErrorLog        string `json:"errorLog"`
	StubStatusURL   string `json:"stubStatusUrl"`
	TimeoutMs       int64  `json:"timeoutMs"`
}

type AgentNginxRequest struct {
	Runtime    AgentRuntimeSpec `json:"runtime"`
	Action     string           `json:"action,omitempty"`
	ConfigPath string           `json:"configPath,omitempty"`
	Config     string           `json:"config,omitempty"`
	TargetPath string           `json:"targetPath,omitempty"`
	Reason     string           `json:"reason,omitempty"`
	LogType    string           `json:"logType,omitempty"`
	Lines      int              `json:"lines,omitempty"`
	Keyword    string           `json:"keyword,omitempty"`
}

func buildAgentRuntime(runtime nginxRuntime) AgentRuntimeSpec {
	return AgentRuntimeSpec{
		InstanceGuid:    runtime.InstanceGuid,
		Mode:            runtime.Mode,
		Bin:             runtime.Bin,
		Systemctl:       runtime.Systemctl,
		ServiceName:     runtime.ServiceName,
		MainConfig:      runtime.MainConfig,
		ManagedConfig:   runtime.ManagedConfig,
		DockerBin:       runtime.DockerBin,
		DockerContainer: runtime.DockerContainer,
		AccessLog:       runtime.AccessLog,
		ErrorLog:        runtime.ErrorLog,
		StubStatusURL:   runtime.StubStatusURL,
		TimeoutMs:       runtime.Timeout.Milliseconds(),
	}
}

func dispatchAgent(runtime nginxRuntime, taskType string, request AgentNginxRequest, response any) error {
	_, err := ServiceGroupApp.AgentService.Dispatch(runtime.NodeGuid, taskType, request, runtime.Timeout+2*time.Second, response)
	return err
}
