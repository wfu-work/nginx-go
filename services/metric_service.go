package services

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"nginx-go/domains"
	"strconv"
	"strings"
	"time"

	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type MetricService struct {
	commonServices.CrudService[domains.MetricSample]
}

type MetricSummary struct {
	InstanceGuid string            `json:"instanceGuid"`
	Status       StatusResult      `json:"status"`
	StubStatus   *StubStatusResult `json:"stubStatus,omitempty"`
	CheckedAt    int64             `json:"checkedAt"`
}

type StubStatusResult struct {
	Active   int `json:"active"`
	Accepts  int `json:"accepts"`
	Handled  int `json:"handled"`
	Requests int `json:"requests"`
	Reading  int `json:"reading"`
	Writing  int `json:"writing"`
	Waiting  int `json:"waiting"`
}

// Summary combines process status and nginx stub_status for one instance.
func (s MetricService) Summary(instanceGuid string) (MetricSummary, error) {
	runtime, err := resolveNginxRuntime(instanceGuid)
	if err != nil {
		return MetricSummary{}, err
	}
	status, err := ServiceGroupApp.NginxService.Status(runtime.InstanceGuid)
	if err != nil {
		return MetricSummary{}, err
	}
	stub, _ := s.StubStatus(runtime.InstanceGuid)
	return MetricSummary{InstanceGuid: runtime.InstanceGuid, Status: status, StubStatus: stub, CheckedAt: time.Now().UnixMilli()}, nil
}

// StubStatus fetches and parses nginx stub_status output.
func (s MetricService) StubStatus(instanceGuid string) (*StubStatusResult, error) {
	runtime, err := resolveNginxRuntime(instanceGuid)
	if err != nil {
		return nil, err
	}
	if runtime.StubStatusURL == "" {
		return nil, errors.New("nginx.stub-status-url is empty")
	}
	if runtime.Remote {
		var result StubStatusResult
		if err := dispatchAgent(runtime, AgentTaskNginxStubStatus, AgentNginxRequest{Runtime: buildAgentRuntime(runtime)}, &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	client := http.Client{Timeout: runtime.Timeout}
	resp, err := client.Get(runtime.StubStatusURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseStubStatus(string(body))
}

// Process returns detected nginx OS processes.
func (s MetricService) Process(instanceGuid string) ([]ProcessStatus, error) {
	runtime, err := resolveNginxRuntime(instanceGuid)
	if err != nil {
		return nil, err
	}
	if runtime.Remote {
		var result []ProcessStatus
		if err := dispatchAgent(runtime, AgentTaskNginxProcess, AgentNginxRequest{Runtime: buildAgentRuntime(runtime)}, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return nginxProcesses(), nil
}

// Collect stores summary metric samples for all enabled instances.
func (s MetricService) Collect() error {
	runtimes, err := listNginxRuntimes()
	if err != nil {
		return err
	}
	var firstErr error
	for _, runtime := range runtimes {
		summary, err := s.Summary(runtime.InstanceGuid)
		status := statusText(err == nil)
		message := ""
		if err != nil {
			message = err.Error()
			if firstErr == nil {
				firstErr = err
			}
		}
		payload, _ := json.Marshal(summary)
		if createErr := s.Create(domains.MetricSample{
			Kind:    "summary",
			Status:  status,
			Payload: string(payload),
			Message: message,
		}); createErr != nil && firstErr == nil {
			firstErr = createErr
		}
	}
	return firstErr
}

// Samples returns paginated metric samples.
func (s MetricService) Samples(params map[string]string) (interface{}, int64, error) {
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Desc == "" && pageInfo.Asc == "" {
		pageInfo.Desc = "createTime"
	}
	return s.List(pageInfo, "kind,status,message")
}

func parseStubStatus(raw string) (*StubStatusResult, error) {
	fields := strings.Fields(raw)
	if len(fields) < 14 {
		return nil, errors.New("invalid nginx stub_status response")
	}
	result := &StubStatusResult{}
	for i, field := range fields {
		switch field {
		case "Active":
			result.Active = atoiSafe(fields[i+2])
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

func atoiSafe(value string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(value))
	return n
}
