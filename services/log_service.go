package services

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"nginx-go/domains"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
	"gorm.io/gorm/clause"
)

type LogService struct {
	accessCrud commonServices.CrudService[domains.AccessLogRecord]
	errorCrud  commonServices.CrudService[domains.ErrorLogRecord]
}

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

type LogSyncResult struct {
	AccessInserted int `json:"accessInserted"`
	ErrorInserted  int `json:"errorInserted"`
}

var (
	accessLogPattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([^"]*)" (\d{3}) (\S+) "([^"]*)" "([^"]*)"`)
	errorLogPattern  = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(\w+)\] .*?: (.*)$`)
)

// Access tails the configured access log for immediate display without requiring database sync.
func (LogService) Access(params map[string]string) (LogResult, error) {
	return readLog(LogQuery{InstanceGuid: params["instanceGuid"], Type: "access", Lines: parseLines(params["lines"]), Keyword: params["keyword"]})
}

// Error tails the configured error log for immediate display without requiring database sync.
func (LogService) Error(params map[string]string) (LogResult, error) {
	return readLog(LogQuery{InstanceGuid: params["instanceGuid"], Type: "error", Lines: parseLines(params["lines"]), Keyword: params["keyword"]})
}

// AccessRecords returns parsed access log records stored by Sync.
func (s LogService) AccessRecords(params map[string]string) (interface{}, int64, error) {
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Desc == "" && pageInfo.Asc == "" {
		pageInfo.Desc = "createTime"
	}
	return s.accessCrud.List(pageInfo, "instanceGuid,remoteAddr,method,path,status,userAgent")
}

// ErrorRecords returns parsed error log records stored by Sync or ScanErrors.
func (s LogService) ErrorRecords(params map[string]string) (interface{}, int64, error) {
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Desc == "" && pageInfo.Asc == "" {
		pageInfo.Desc = "createTime"
	}
	return s.errorCrud.List(pageInfo, "instanceGuid,level,message")
}

// Sync parses recent access/error log lines into database records for filtering and aggregation.
func (s LogService) Sync(params map[string]string) (LogSyncResult, error) {
	runtime, err := resolveNginxRuntime(params["instanceGuid"])
	if err != nil {
		return LogSyncResult{}, err
	}
	limit := parseLines(params["lines"])
	accessInserted, accessErr := s.syncAccess(runtime, limit)
	errorInserted, errorErr := s.syncError(runtime, limit)
	result := LogSyncResult{AccessInserted: accessInserted, ErrorInserted: errorInserted}
	if accessErr != nil {
		return result, accessErr
	}
	return result, errorErr
}

// ScanErrors scans all enabled instances for critical error-log lines and stores alert samples.
func (LogService) ScanErrors() error {
	runtimes, err := listNginxRuntimes()
	if err != nil {
		return err
	}
	var firstErr error
	for _, runtime := range runtimes {
		lines, err := tailRuntimeLog(runtime, "error", 500, "")
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
		_, _ = LogService{}.saveErrorRecords(runtime, matches)
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
		ServiceGroupApp.EventNotificationService.Notify(EventNotificationCreate{
			Title:      "Nginx 错误日志告警",
			Content:    fmt.Sprintf("%s 检测到 %d 条关键错误", runtime.InstanceGuid, len(matches)),
			Level:      domains.EventNotificationLevelWarning,
			SourceType: "error_log",
			SourceGuid: runtime.InstanceGuid,
		})
	}
	return firstErr
}

func (s LogService) syncAccess(runtime nginxRuntime, limit int) (int, error) {
	lines, err := tailRuntimeLog(runtime, "access", limit, "")
	if err != nil {
		return 0, err
	}
	return s.saveAccessRecords(runtime, lines)
}

func (s LogService) syncError(runtime nginxRuntime, limit int) (int, error) {
	lines, err := tailRuntimeLog(runtime, "error", limit, "")
	if err != nil {
		return 0, err
	}
	return s.saveErrorRecords(runtime, lines)
}

func (s LogService) saveAccessRecords(runtime nginxRuntime, lines []string) (int, error) {
	inserted := 0
	for _, line := range lines {
		record, ok := parseAccessLogLine(runtime.InstanceGuid, line)
		if !ok {
			continue
		}
		result := s.accessCrud.DB().Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
		if result.Error != nil {
			return inserted, result.Error
		}
		if result.RowsAffected > 0 {
			inserted++
		}
	}
	return inserted, nil
}

func (s LogService) saveErrorRecords(runtime nginxRuntime, lines []string) (int, error) {
	inserted := 0
	for _, line := range lines {
		record := parseErrorLogLine(runtime.InstanceGuid, line)
		result := s.errorCrud.DB().Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
		if result.Error != nil {
			return inserted, result.Error
		}
		if result.RowsAffected > 0 {
			inserted++
		}
	}
	return inserted, nil
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
	lines, err := tailRuntimeLog(runtime, query.Type, query.Lines, query.Keyword)
	if err != nil {
		return LogResult{}, err
	}
	return LogResult{Path: path, Lines: lines}, nil
}

func tailRuntimeLog(runtime nginxRuntime, kind string, limit int, keyword string) ([]string, error) {
	if runtime.Remote {
		var result LogResult
		err := dispatchAgent(runtime, AgentTaskNginxLogTail, AgentNginxRequest{
			Runtime: buildAgentRuntime(runtime),
			LogType: kind,
			Lines:   limit,
			Keyword: keyword,
		}, &result)
		return result.Lines, err
	}
	return tailFile(logPath(runtime, kind), limit, keyword)
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

func parseAccessLogLine(instanceGuid, line string) (domains.AccessLogRecord, bool) {
	matches := accessLogPattern.FindStringSubmatch(line)
	if len(matches) == 0 {
		return domains.AccessLogRecord{}, false
	}
	method, path, protocol := splitRequest(matches[3])
	bytesSent, _ := strconv.ParseInt(matches[5], 10, 64)
	if matches[5] == "-" {
		bytesSent = 0
	}
	status, _ := strconv.Atoi(matches[4])
	return domains.AccessLogRecord{
		InstanceGuid:  instanceGuid,
		RemoteAddr:    matches[1],
		TimeLocal:     matches[2],
		Method:        method,
		Path:          path,
		Protocol:      protocol,
		Status:        status,
		BodyBytesSent: bytesSent,
		Referer:       matches[6],
		UserAgent:     matches[7],
		RawLine:       line,
		LineHash:      lineHash(instanceGuid, "access", line),
	}, true
}

func parseErrorLogLine(instanceGuid, line string) domains.ErrorLogRecord {
	record := domains.ErrorLogRecord{
		InstanceGuid: instanceGuid,
		Level:        "unknown",
		Message:      line,
		RawLine:      line,
		LineHash:     lineHash(instanceGuid, "error", line),
	}
	matches := errorLogPattern.FindStringSubmatch(line)
	if len(matches) > 0 {
		record.TimeLocal = matches[1]
		record.Level = matches[2]
		record.Message = matches[3]
	}
	return record
}

func splitRequest(request string) (string, string, string) {
	parts := strings.Fields(request)
	if len(parts) != 3 {
		return "", request, ""
	}
	return parts[0], parts[1], parts[2]
}

func lineHash(instanceGuid, kind, line string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", instanceGuid, kind, line)))
	return fmt.Sprintf("%x", sum)
}
