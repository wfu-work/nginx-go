package services

import (
	"encoding/json"
	"errors"
	"net"
	"nginx-go/domains"
	"time"

	"github.com/wfu-work/nav-common-go-lib/global"
	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type UpstreamService struct {
	commonServices.CrudService[domains.Upstream]
	serverCrud commonServices.CrudService[domains.UpstreamServer]
}

type UpstreamHealthResult struct {
	UpstreamGuid string                 `json:"upstreamGuid"`
	Name         string                 `json:"name"`
	Healthy      bool                   `json:"healthy"`
	CheckedAt    int64                  `json:"checkedAt"`
	Servers      []UpstreamServerHealth `json:"servers"`
}

type UpstreamServerHealth struct {
	ServerGuid string `json:"serverGuid"`
	Address    string `json:"address"`
	Healthy    bool   `json:"healthy"`
	Message    string `json:"message"`
	LatencyMs  int64  `json:"latencyMs"`
}

// List returns paginated upstream groups.
func (s UpstreamService) List(params map[string]string) (interface{}, int64, error) {
	return s.CrudService.List(commonUtils.ToPageInfo(params), "name,method")
}

// Create stores an upstream group.
func (s UpstreamService) Create(upstream domains.Upstream) error {
	return s.CrudService.Create(upstream)
}

// Update updates an upstream group by guid.
func (s UpstreamService) Update(guid string, upstream domains.Upstream) error {
	if guid == "" {
		return errors.New("missing upstream guid")
	}
	upstream.Guid = guid
	return s.CrudService.Updates(upstream)
}

// Delete soft-deletes an upstream group by guid.
func (s UpstreamService) Delete(guid string) error {
	if guid == "" {
		return errors.New("missing upstream guid")
	}
	return s.CrudService.DeleteByGuid(guid)
}

// Get returns an upstream group together with its servers.
func (s UpstreamService) Get(guid string) (map[string]any, error) {
	upstream, err := s.CrudService.GetByGuid(guid)
	if err != nil {
		return nil, err
	}
	if upstream == nil {
		return nil, errors.New("upstream not found")
	}
	servers, err := s.serverCrud.ListByFields(map[string]any{"upstreamGuid": guid})
	if err != nil {
		return nil, err
	}
	return map[string]any{"upstream": upstream, "servers": servers}, nil
}

// Health checks TCP connectivity for every server in one upstream group.
func (s UpstreamService) Health(guid string) (UpstreamHealthResult, error) {
	if guid == "" {
		return UpstreamHealthResult{}, errors.New("missing upstream guid")
	}
	upstream, err := s.CrudService.GetByGuid(guid)
	if err != nil {
		return UpstreamHealthResult{}, err
	}
	if upstream == nil {
		return UpstreamHealthResult{}, errors.New("upstream not found")
	}
	servers, err := s.serverCrud.ListByFields(map[string]any{"upstreamGuid": guid})
	if err != nil {
		return UpstreamHealthResult{}, err
	}
	result := UpstreamHealthResult{
		UpstreamGuid: guid,
		Name:         upstream.Name,
		Healthy:      len(servers) > 0,
		CheckedAt:    time.Now().UnixMilli(),
		Servers:      make([]UpstreamServerHealth, 0, len(servers)),
	}
	for _, server := range servers {
		health := checkUpstreamServer(server)
		if !health.Healthy {
			result.Healthy = false
		}
		result.Servers = append(result.Servers, health)
	}
	return result, nil
}

// CollectHealth stores upstream health snapshots as metric samples for dashboard use.
func (s UpstreamService) CollectHealth() error {
	var upstreams []domains.Upstream
	if err := global.NAV_DB.Order("id asc").Find(&upstreams).Error; err != nil {
		return err
	}
	var firstErr error
	for _, upstream := range upstreams {
		result, err := s.Health(upstream.Guid)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		payload, _ := json.Marshal(result)
		message := "upstream health check success"
		if !result.Healthy {
			message = "upstream health check failed"
		}
		if createErr := ServiceGroupApp.MetricService.Create(domains.MetricSample{
			Kind:    "upstream_health",
			Status:  statusText(result.Healthy),
			Payload: string(payload),
			Message: message,
		}); createErr != nil && firstErr == nil {
			firstErr = createErr
		}
		if !result.Healthy {
			ServiceGroupApp.EventNotificationService.Notify(EventNotificationCreate{
				Title:      "上游健康检查异常",
				Content:    result.Name + " 存在不可用节点",
				Level:      domains.EventNotificationLevelWarning,
				SourceType: "upstream",
				SourceGuid: result.UpstreamGuid,
			})
		}
	}
	return firstErr
}

// CreateServer stores an upstream server node after applying default nginx values.
func (s UpstreamService) CreateServer(server domains.UpstreamServer) error {
	defaultUpstreamServer(&server)
	return s.serverCrud.Create(server)
}

// UpdateServer updates one upstream server node by guid.
func (s UpstreamService) UpdateServer(guid string, server domains.UpstreamServer) error {
	if guid == "" {
		return errors.New("missing upstream server guid")
	}
	server.Guid = guid
	defaultUpstreamServer(&server)
	return s.serverCrud.Updates(server)
}

// DeleteServer soft-deletes one upstream server node by guid.
func (s UpstreamService) DeleteServer(guid string) error {
	if guid == "" {
		return errors.New("missing upstream server guid")
	}
	return s.serverCrud.DeleteByGuid(guid)
}

func defaultUpstreamServer(server *domains.UpstreamServer) {
	if server.Weight == 0 {
		server.Weight = 1
	}
	if server.MaxFails == 0 {
		server.MaxFails = 3
	}
	if server.FailTimeout == "" {
		server.FailTimeout = "30s"
	}
}

func checkUpstreamServer(server domains.UpstreamServer) UpstreamServerHealth {
	start := time.Now()
	if server.Down {
		return UpstreamServerHealth{
			ServerGuid: server.Guid,
			Address:    server.Address,
			Healthy:    false,
			Message:    "server is marked down",
		}
	}
	address := normalizeUpstreamAddress(server.Address)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return UpstreamServerHealth{
			ServerGuid: server.Guid,
			Address:    server.Address,
			Healthy:    false,
			Message:    err.Error(),
			LatencyMs:  latency,
		}
	}
	_ = conn.Close()
	return UpstreamServerHealth{
		ServerGuid: server.Guid,
		Address:    server.Address,
		Healthy:    true,
		Message:    "tcp connect success",
		LatencyMs:  latency,
	}
}

func normalizeUpstreamAddress(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err == nil && host != "" && port != "" {
		return address
	}
	if address == "" {
		return address
	}
	return net.JoinHostPort(address, "80")
}
