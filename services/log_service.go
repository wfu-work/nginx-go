package services

import (
	"bufio"
	"encoding/json"
	"errors"
	"nginx-go/domains"
	"os"
	"strings"
	"time"
)

type LogService struct{}

type LogQuery struct {
	InstanceGuid string `json:"instanceGuid" form:"instanceGuid"`
	Type         string `json:"type" form:"type"`
	Lines        int    `json:"lines" form:"lines"`
	Keyword      string `json:"keyword" form:"keyword"`
}

type LogResult struct {
	Path  string   `json:"path"`
	Lines []string `json:"lines"`
}

type ErrorLogAlert struct {
	InstanceGuid string   `json:"instanceGuid"`
	Path         string   `json:"path"`
	Lines        []string `json:"lines"`
	CheckedAt    int64    `json:"checkedAt"`
}

func (LogService) Access(params map[string]string) (LogResult, error) {
	return readLog(LogQuery{InstanceGuid: params["instanceGuid"], Type: "access", Lines: parseLines(params["lines"]), Keyword: params["keyword"]})
}

func (LogService) Error(params map[string]string) (LogResult, error) {
	return readLog(LogQuery{InstanceGuid: params["instanceGuid"], Type: "error", Lines: parseLines(params["lines"]), Keyword: params["keyword"]})
}

func (LogService) ScanErrors() error {
	runtimes, err := listNginxRuntimes()
	if err != nil {
		return err
	}
	var firstErr error
	for _, runtime := range runtimes {
		lines, err := tailFile(runtime.ErrorLog, 500, "")
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		matches := filterErrorLines(lines)
		if len(matches) == 0 {
			continue
		}
		alert := ErrorLogAlert{
			InstanceGuid: runtime.InstanceGuid,
			Path:         runtime.ErrorLog,
			Lines:        matches,
			CheckedAt:    time.Now().UnixMilli(),
		}
		payload, _ := json.Marshal(alert)
		if createErr := ServiceGroupApp.MetricService.Create(domains.MetricSample{
			Kind:    "error_log_alert",
			Status:  "warning",
			Payload: string(payload),
			Message: "nginx error log contains critical keywords",
		}); createErr != nil && firstErr == nil {
			firstErr = createErr
		}
	}
	return firstErr
}

func readLog(query LogQuery) (LogResult, error) {
	runtime, err := resolveNginxRuntime(query.InstanceGuid)
	if err != nil {
		return LogResult{}, err
	}
	path := logPath(runtime, query.Type)
	if path == "" {
		return LogResult{}, errors.New("log path is empty")
	}
	lines, err := tailFile(path, query.Lines, query.Keyword)
	if err != nil {
		return LogResult{}, err
	}
	return LogResult{Path: path, Lines: lines}, nil
}

func logPath(runtime nginxRuntime, kind string) string {
	if kind == "error" {
		return runtime.ErrorLog
	}
	return runtime.AccessLog
}

func parseLines(value string) int {
	n := atoiSafe(value)
	if n <= 0 {
		return 200
	}
	if n > 2000 {
		return 2000
	}
	return n
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

func filterErrorLines(lines []string) []string {
	keywords := []string{"[error]", "[crit]", "[alert]", "[emerg]"}
	matches := make([]string, 0)
	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				matches = append(matches, line)
				break
			}
		}
	}
	if len(matches) > 50 {
		return matches[len(matches)-50:]
	}
	return matches
}
