package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

const (
	taskNginxStatus         = "nginx.status"
	taskNginxOperation      = "nginx.operation"
	taskNginxStubStatus     = "nginx.stub_status"
	taskNginxProcess        = "nginx.process"
	taskNginxLogTail        = "nginx.log_tail"
	taskNginxConfigValidate = "nginx.config_validate"
	taskNginxConfigPublish  = "nginx.config_publish"
)

type config struct {
	CenterURL string
	Token     string
	AgentID   string
	Name      string
	Address   string
	Labels    string
	Version   string
	Interval  time.Duration
}

type registerRequest struct {
	AgentID string `json:"agentId"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Labels  string `json:"labels"`
	Version string `json:"version"`
}

type node struct {
	Guid string `json:"guid"`
}

type apiResponse[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type taskEnvelope struct {
	Guid      string          `json:"guid"`
	NodeGuid  string          `json:"nodeGuid"`
	TaskType  string          `json:"taskType"`
	Request   json.RawMessage `json:"request"`
	TimeoutMs int64           `json:"timeoutMs"`
}

type taskCompleteRequest struct {
	Success  bool   `json:"success"`
	Response any    `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

type runtimeSpec struct {
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

type nginxRequest struct {
	Runtime    runtimeSpec `json:"runtime"`
	Action     string      `json:"action"`
	ConfigPath string      `json:"configPath"`
	Config     string      `json:"config"`
	TargetPath string      `json:"targetPath"`
	Reason     string      `json:"reason"`
	LogType    string      `json:"logType"`
	Lines      int         `json:"lines"`
	Keyword    string      `json:"keyword"`
}

type commandResult struct {
	Command    string `json:"command"`
	Output     string `json:"output"`
	Success    bool   `json:"success"`
	DurationMs int64  `json:"durationMs"`
}

type processStatus struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpuPercent"`
	MemoryMB   float32 `json:"memoryMb"`
}

type statusResult struct {
	InstanceGuid string          `json:"instanceGuid"`
	Mode         string          `json:"mode"`
	Running      bool            `json:"running"`
	Message      string          `json:"message"`
	Version      string          `json:"version"`
	Processes    []processStatus `json:"processes"`
	CheckedAt    int64           `json:"checkedAt"`
}

type stubStatusResult struct {
	Active   int `json:"active"`
	Accepts  int `json:"accepts"`
	Handled  int `json:"handled"`
	Requests int `json:"requests"`
	Reading  int `json:"reading"`
	Writing  int `json:"writing"`
	Waiting  int `json:"waiting"`
}

type logResult struct {
	Path  string   `json:"path"`
	Lines []string `json:"lines"`
}

type validateResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Output     string `json:"output"`
	ConfigPath string `json:"configPath"`
	DurationMs int64  `json:"durationMs"`
}

type publishResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	TargetPath string `json:"targetPath"`
	BackupPath string `json:"backupPath"`
	DurationMs int64  `json:"durationMs"`
}

func main() {
	cfg := parseConfig()
	client := http.Client{Timeout: 30 * time.Second}
	registered, err := register(client, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "register failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("nginx-agent registered: node=%s center=%s\n", registered.Guid, cfg.CenterURL)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		if err := heartbeat(client, cfg, registered.Guid); err != nil {
			fmt.Fprintf(os.Stderr, "heartbeat failed: %v\n", err)
		}
		tasks, err := pollTasks(client, cfg, registered.Guid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "poll failed: %v\n", err)
		}
		for _, task := range tasks {
			if err := handleTask(client, cfg, task); err != nil {
				fmt.Fprintf(os.Stderr, "task %s failed: %v\n", task.Guid, err)
			}
		}
		<-ticker.C
	}
}

func parseConfig() config {
	cfg := config{}
	flag.StringVar(&cfg.CenterURL, "center", env("NGINX_AGENT_CENTER", "http://127.0.0.1:3007/api"), "center API base URL")
	flag.StringVar(&cfg.Token, "token", env("NGINX_AGENT_TOKEN", ""), "shared agent token")
	flag.StringVar(&cfg.AgentID, "agent-id", env("NGINX_AGENT_ID", hostname()), "stable agent ID")
	flag.StringVar(&cfg.Name, "name", env("NGINX_AGENT_NAME", hostname()), "node display name")
	flag.StringVar(&cfg.Address, "address", env("NGINX_AGENT_ADDRESS", ""), "node address")
	flag.StringVar(&cfg.Labels, "labels", env("NGINX_AGENT_LABELS", ""), "node labels")
	flag.StringVar(&cfg.Version, "version", env("NGINX_AGENT_VERSION", "0.1.0"), "agent version")
	interval := flag.Int("interval", envInt("NGINX_AGENT_INTERVAL_SECONDS", 3), "poll interval in seconds")
	flag.Parse()
	cfg.CenterURL = strings.TrimRight(cfg.CenterURL, "/")
	cfg.Interval = time.Duration(*interval) * time.Second
	if cfg.Interval <= 0 {
		cfg.Interval = 3 * time.Second
	}
	return cfg
}

func register(client http.Client, cfg config) (node, error) {
	req := registerRequest{AgentID: cfg.AgentID, Name: cfg.Name, Address: cfg.Address, Labels: cfg.Labels, Version: cfg.Version}
	return post[node](client, cfg, "/agent/register", req)
}

func heartbeat(client http.Client, cfg config, nodeGuid string) error {
	_, err := post[node](client, cfg, "/agent/heartbeat", map[string]string{
		"nodeGuid": nodeGuid,
		"agentId":  cfg.AgentID,
		"address":  cfg.Address,
		"version":  cfg.Version,
	})
	return err
}

func pollTasks(client http.Client, cfg config, nodeGuid string) ([]taskEnvelope, error) {
	return get[[]taskEnvelope](client, cfg, "/agent/tasks/poll?nodeGuid="+nodeGuid)
}

func handleTask(client http.Client, cfg config, task taskEnvelope) error {
	result, err := executeTask(task)
	complete := taskCompleteRequest{Success: err == nil, Response: result}
	if err != nil {
		complete.Error = err.Error()
	}
	_, completeErr := post[map[string]any](client, cfg, "/agent/tasks/"+task.Guid+"/complete", complete)
	return completeErr
}

func executeTask(task taskEnvelope) (any, error) {
	var req nginxRequest
	if err := json.Unmarshal(task.Request, &req); err != nil {
		return nil, err
	}
	switch task.TaskType {
	case taskNginxStatus:
		return status(req.Runtime), nil
	case taskNginxOperation:
		return operation(req.Runtime, req.Action, req.ConfigPath, req.Config), nil
	case taskNginxStubStatus:
		return stubStatus(req.Runtime)
	case taskNginxProcess:
		return nginxProcesses(), nil
	case taskNginxLogTail:
		return tailLog(req.Runtime, req.LogType, req.Lines, req.Keyword)
	case taskNginxConfigValidate:
		return validateConfig(req.Runtime, req.Config)
	case taskNginxConfigPublish:
		return publishConfig(req.Runtime, req.Config, req.TargetPath)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.TaskType)
	}
}

func status(runtime runtimeSpec) statusResult {
	processes := nginxProcesses()
	version := runNginx(runtime, "-v")
	running := nginxIsRunning(runtime, processes)
	message := "nginx status refreshed"
	if !running {
		message = "nginx is not running or cannot be detected"
	}
	return statusResult{
		InstanceGuid: runtime.InstanceGuid,
		Mode:         runtime.Mode,
		Running:      running,
		Message:      message,
		Version:      version.Output,
		Processes:    processes,
		CheckedAt:    time.Now().UnixMilli(),
	}
}

func operation(runtime runtimeSpec, action, configPath, config string) commandResult {
	if action == "test" {
		path := configPath
		if config != "" {
			var cleanup func()
			generated, cleanupFn, err := writeTempConfig(config)
			if err != nil {
				return commandResult{Success: false, Output: err.Error()}
			}
			path = generated
			cleanup = cleanupFn
			defer cleanup()
		}
		args := []string{"-t"}
		if path != "" {
			args = append(args, "-c", path)
		} else if runtime.MainConfig != "" {
			args = append(args, "-c", runtime.MainConfig)
		}
		return runNginx(runtime, args...)
	}
	switch runtime.Mode {
	case "systemd":
		return runCommand(runtime, runtime.Systemctl, action, runtime.ServiceName)
	case "docker":
		return runDockerMode(runtime, action)
	default:
		return runCommandMode(runtime, action)
	}
}

func validateConfig(runtime runtimeSpec, config string) (validateResult, error) {
	path, cleanup, err := writeTempConfig(config)
	if err != nil {
		return validateResult{}, err
	}
	defer cleanup()
	cmd := operation(runtime, "test", path, "")
	result := validateResult{
		Success:    cmd.Success,
		Message:    "nginx config test success",
		Output:     cmd.Output,
		ConfigPath: path,
		DurationMs: cmd.DurationMs,
	}
	if !cmd.Success {
		result.Message = "nginx config test failed"
	}
	return result, nil
}

func publishConfig(runtime runtimeSpec, config, targetPath string) (publishResult, error) {
	start := time.Now()
	if targetPath == "" {
		targetPath = runtime.ManagedConfig
	}
	if targetPath == "" {
		return publishResult{}, errors.New("managed config path is empty")
	}
	validate, err := validateConfig(runtime, config)
	if err != nil {
		return publishResult{Success: false, Message: err.Error(), TargetPath: targetPath, DurationMs: time.Since(start).Milliseconds()}, err
	}
	if !validate.Success {
		return publishResult{Success: false, Message: validate.Message, TargetPath: targetPath, DurationMs: time.Since(start).Milliseconds()}, nil
	}
	backupPath, err := backupExistingConfig(targetPath)
	if err != nil {
		return publishResult{Success: false, Message: err.Error(), TargetPath: targetPath, DurationMs: time.Since(start).Milliseconds()}, err
	}
	if err := atomicWriteFile(targetPath, []byte(config), 0644); err != nil {
		return publishResult{Success: false, Message: err.Error(), TargetPath: targetPath, BackupPath: backupPath, DurationMs: time.Since(start).Milliseconds()}, err
	}
	reload := operation(runtime, "reload", targetPath, "")
	if reload.Success {
		return publishResult{Success: true, Message: "nginx config publish success", TargetPath: targetPath, BackupPath: backupPath, DurationMs: time.Since(start).Milliseconds()}, nil
	}
	if backupPath != "" {
		_ = restoreBackup(targetPath, backupPath)
	}
	return publishResult{Success: false, Message: reload.Output, TargetPath: targetPath, BackupPath: backupPath, DurationMs: time.Since(start).Milliseconds()}, nil
}

func stubStatus(runtime runtimeSpec) (stubStatusResult, error) {
	if runtime.StubStatusURL == "" {
		return stubStatusResult{}, errors.New("stub status url is empty")
	}
	client := http.Client{Timeout: timeout(runtime)}
	resp, err := client.Get(runtime.StubStatusURL)
	if err != nil {
		return stubStatusResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return stubStatusResult{}, err
	}
	return parseStubStatus(string(body))
}

func tailLog(runtime runtimeSpec, kind string, limit int, keyword string) (logResult, error) {
	path := runtime.AccessLog
	if kind == "error" {
		path = runtime.ErrorLog
	}
	if path == "" {
		return logResult{}, errors.New("log path is empty")
	}
	lines, err := tailFile(path, normalizeLines(limit), keyword)
	return logResult{Path: path, Lines: lines}, err
}

func runCommandMode(runtime runtimeSpec, action string) commandResult {
	switch action {
	case "reload":
		return runCommand(runtime, runtime.Bin, "-s", "reload")
	case "stop":
		return runCommand(runtime, runtime.Bin, "-s", "stop")
	case "start":
		return runCommand(runtime, runtime.Bin)
	case "restart":
		stop := runCommand(runtime, runtime.Bin, "-s", "stop")
		if !stop.Success {
			return stop
		}
		return runCommand(runtime, runtime.Bin)
	default:
		return commandResult{Success: false, Output: "unsupported command action"}
	}
}

func runDockerMode(runtime runtimeSpec, action string) commandResult {
	if runtime.DockerContainer == "" {
		return commandResult{Success: false, Output: "nginx docker container is empty"}
	}
	switch action {
	case "reload":
		return runCommand(runtime, runtime.DockerBin, "exec", runtime.DockerContainer, runtime.Bin, "-s", "reload")
	case "stop":
		return runCommand(runtime, runtime.DockerBin, "stop", runtime.DockerContainer)
	case "start":
		return runCommand(runtime, runtime.DockerBin, "start", runtime.DockerContainer)
	case "restart":
		return runCommand(runtime, runtime.DockerBin, "restart", runtime.DockerContainer)
	default:
		return commandResult{Success: false, Output: "unsupported docker action"}
	}
}

func runNginx(runtime runtimeSpec, args ...string) commandResult {
	if runtime.Mode == "docker" {
		if runtime.DockerContainer == "" {
			return commandResult{Success: false, Output: "nginx docker container is empty"}
		}
		dockerArgs := append([]string{"exec", runtime.DockerContainer, runtime.Bin}, args...)
		return runCommand(runtime, runtime.DockerBin, dockerArgs...)
	}
	return runCommand(runtime, runtime.Bin, args...)
}

func runCommand(runtime runtimeSpec, name string, args ...string) commandResult {
	start := time.Now()
	if name == "" {
		return commandResult{Success: false, Output: "command is empty"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout(runtime))
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	result := commandResult{
		Command:    strings.TrimSpace(name + " " + strings.Join(args, " ")),
		Output:     strings.TrimSpace(string(output)),
		Success:    err == nil,
		DurationMs: time.Since(start).Milliseconds(),
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.Output = strings.TrimSpace(result.Output + "\ncommand timeout")
	}
	if err != nil && result.Output == "" {
		result.Output = err.Error()
	}
	return result
}

func nginxProcesses() []processStatus {
	items, err := process.Processes()
	if err != nil {
		return nil
	}
	results := make([]processStatus, 0)
	for _, item := range items {
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
		results = append(results, processStatus{PID: item.Pid, Name: name, CPUPercent: cpuPercent, MemoryMB: memoryMB})
	}
	return results
}

func nginxIsRunning(runtime runtimeSpec, processes []processStatus) bool {
	switch runtime.Mode {
	case "systemd":
		result := runCommand(runtime, runtime.Systemctl, "is-active", runtime.ServiceName)
		return result.Success && strings.TrimSpace(result.Output) == "active" || len(processes) > 0
	case "docker":
		if runtime.DockerContainer == "" {
			return false
		}
		result := runCommand(runtime, runtime.DockerBin, "inspect", "-f", "{{.State.Running}}", runtime.DockerContainer)
		return result.Success && strings.TrimSpace(result.Output) == "true"
	default:
		return len(processes) > 0
	}
}

func parseStubStatus(raw string) (stubStatusResult, error) {
	fields := strings.Fields(raw)
	if len(fields) < 14 {
		return stubStatusResult{}, errors.New("invalid nginx stub_status response")
	}
	result := stubStatusResult{}
	for i, field := range fields {
		switch field {
		case "Active":
			if i+2 < len(fields) {
				result.Active = atoiSafe(fields[i+2])
			}
		case "server":
			if i+6 < len(fields) {
				result.Accepts = atoiSafe(fields[i+4])
				result.Handled = atoiSafe(fields[i+5])
				result.Requests = atoiSafe(fields[i+6])
			}
		case "Reading:":
			if i+5 < len(fields) {
				result.Reading = atoiSafe(fields[i+1])
				result.Writing = atoiSafe(fields[i+3])
				result.Waiting = atoiSafe(fields[i+5])
			}
		}
	}
	return result, nil
}

func tailFile(path string, limit int, keyword string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	buffer := make([]string, 0, limit)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if keyword != "" && !strings.Contains(line, keyword) {
			continue
		}
		if len(buffer) >= limit {
			copy(buffer, buffer[1:])
			buffer[len(buffer)-1] = line
		} else {
			buffer = append(buffer, line)
		}
	}
	return buffer, scanner.Err()
}

func backupExistingConfig(targetPath string) (string, error) {
	if _, err := os.Stat(targetPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	backupDir := filepath.Join(filepath.Dir(targetPath), "backups")
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

func writeTempConfig(config string) (string, func(), error) {
	dir := filepath.Join(os.TempDir(), "nginx-agent")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", func() {}, err
	}
	path := filepath.Join(dir, fmt.Sprintf("nginx-%d.conf", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		return "", func() {}, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func post[T any](client http.Client, cfg config, path string, body any) (T, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		var zero T
		return zero, err
	}
	req, err := http.NewRequest(http.MethodPost, cfg.CenterURL+path, bytes.NewReader(payload))
	if err != nil {
		var zero T
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("X-Agent-Token", cfg.Token)
	}
	return do[T](client, req)
}

func get[T any](client http.Client, cfg config, path string) (T, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.CenterURL+path, nil)
	if err != nil {
		var zero T
		return zero, err
	}
	if cfg.Token != "" {
		req.Header.Set("X-Agent-Token", cfg.Token)
	}
	return do[T](client, req)
}

func do[T any](client http.Client, req *http.Request) (T, error) {
	var zero T
	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	if resp.StatusCode >= 300 {
		return zero, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var envelope apiResponse[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return zero, err
	}
	if envelope.Code != 0 && envelope.Code != 200 {
		if envelope.Msg == "" {
			envelope.Msg = "request failed"
		}
		return zero, errors.New(envelope.Msg)
	}
	return envelope.Data, nil
}

func timeout(runtime runtimeSpec) time.Duration {
	if runtime.TimeoutMs <= 0 {
		return 10 * time.Second
	}
	return time.Duration(runtime.TimeoutMs) * time.Millisecond
}

func normalizeLines(value int) int {
	if value <= 0 {
		return 200
	}
	if value > 2000 {
		return 2000
	}
	return value
}

func atoiSafe(value string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(value))
	return n
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "nginx-agent"
	}
	return regexp.MustCompile(`[^a-zA-Z0-9_.-]+`).ReplaceAllString(name, "-")
}
