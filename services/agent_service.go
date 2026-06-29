package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"nginx-go/domains"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	"go.uber.org/zap"
)

type AgentService struct {
	taskCrud commonServices.CrudService[domains.AgentTask]
}

type AgentRegisterRequest struct {
	AgentID     string `json:"agentId"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Labels      string `json:"labels"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type AgentHeartbeatRequest struct {
	NodeGuid string `json:"nodeGuid"`
	AgentID  string `json:"agentId"`
	Address  string `json:"address"`
	Version  string `json:"version"`
}

type AgentTaskCompleteRequest struct {
	Success  bool            `json:"success"`
	Response json.RawMessage `json:"response"`
	Error    string          `json:"error"`
}

type AgentTaskEnvelope struct {
	Guid      string          `json:"guid"`
	NodeGuid  string          `json:"nodeGuid"`
	TaskType  string          `json:"taskType"`
	Request   json.RawMessage `json:"request"`
	TimeoutMs int64           `json:"timeoutMs"`
}

type agentWaiterHub struct {
	mu      sync.Mutex
	waiters map[string]chan domains.AgentTask
}

var agentWaiters = &agentWaiterHub{waiters: make(map[string]chan domains.AgentTask)}

func (s AgentService) Register(req AgentRegisterRequest) (*domains.Node, error) {
	if req.AgentID == "" {
		req.AgentID = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	now := time.Now().UnixMilli()
	var existing domains.Node
	result := global.NAV_DB.Where("agent_id = ?", req.AgentID).First(&existing)
	if result.Error == nil && result.RowsAffected > 0 {
		updates := map[string]any{
			"name":         fallback(req.Name, existing.Name),
			"address":      fallback(req.Address, existing.Address),
			"labels":       fallback(req.Labels, existing.Labels),
			"version":      fallback(req.Version, existing.Version),
			"description":  fallback(req.Description, existing.Description),
			"access_mode":  domains.NodeAccessAgent,
			"status":       domains.NodeStatusOnline,
			"enabled":      true,
			"last_seen_at": now,
		}
		if err := global.NAV_DB.Model(&domains.Node{}).Where("guid = ?", existing.Guid).Updates(updates).Error; err != nil {
			return nil, err
		}
		updated, err := ServiceGroupApp.NodeService.Get(existing.Guid)
		if err != nil {
			return nil, err
		}
		return updated, nil
	}
	node := domains.Node{
		BaseDataEntity: commonDomains.BaseDataEntity{Guid: strings.ReplaceAll(uuid.NewString(), "-", "")},
		Name:           fallback(req.Name, req.AgentID),
		AccessMode:     domains.NodeAccessAgent,
		AgentID:        req.AgentID,
		Address:        req.Address,
		Labels:         req.Labels,
		Status:         domains.NodeStatusOnline,
		Version:        req.Version,
		LastSeenAt:     now,
		Enabled:        true,
		Description:    req.Description,
	}
	if err := ServiceGroupApp.NodeService.Create(node); err != nil {
		return nil, err
	}
	return &node, nil
}

func (s AgentService) Heartbeat(req AgentHeartbeatRequest) (*domains.Node, error) {
	node, err := s.resolveNode(req.NodeGuid, req.AgentID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"status":       domains.NodeStatusOnline,
		"last_seen_at": time.Now().UnixMilli(),
	}
	if req.Address != "" {
		updates["address"] = req.Address
	}
	if req.Version != "" {
		updates["version"] = req.Version
	}
	if err := global.NAV_DB.Model(&domains.Node{}).Where("guid = ?", node.Guid).Updates(updates).Error; err != nil {
		return nil, err
	}
	return ServiceGroupApp.NodeService.Get(node.Guid)
}

func (s AgentService) Poll(nodeGuid, agentID string, limit int) ([]AgentTaskEnvelope, error) {
	node, err := s.resolveNode(nodeGuid, agentID)
	if err != nil {
		return nil, err
	}
	if !node.Enabled {
		return nil, errors.New("node is disabled")
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	now := time.Now().UnixMilli()
	_ = global.NAV_DB.Model(&domains.Node{}).Where("guid = ?", node.Guid).Updates(map[string]any{
		"status":       domains.NodeStatusOnline,
		"last_seen_at": now,
	}).Error
	var tasks []domains.AgentTask
	if err := global.NAV_DB.
		Where("node_guid = ? AND status = ?", node.Guid, domains.AgentTaskStatusPending).
		Order("id asc").
		Limit(limit).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	envelopes := make([]AgentTaskEnvelope, 0, len(tasks))
	for _, task := range tasks {
		if err := global.NAV_DB.Model(&domains.AgentTask{}).Where("guid = ? AND status = ?", task.Guid, domains.AgentTaskStatusPending).Updates(map[string]any{
			"status":        domains.AgentTaskStatusDispatched,
			"dispatched_at": now,
		}).Error; err != nil {
			return nil, err
		}
		envelopes = append(envelopes, AgentTaskEnvelope{
			Guid:      task.Guid,
			NodeGuid:  task.NodeGuid,
			TaskType:  task.TaskType,
			Request:   json.RawMessage(task.Request),
			TimeoutMs: task.TimeoutMs,
		})
	}
	return envelopes, nil
}

func (s AgentService) Complete(taskGuid string, req AgentTaskCompleteRequest) (*domains.AgentTask, error) {
	if taskGuid == "" {
		return nil, errors.New("missing task guid")
	}
	status := domains.AgentTaskStatusSuccess
	if !req.Success {
		status = domains.AgentTaskStatusFailed
	}
	responsePayload := string(req.Response)
	updates := map[string]any{
		"status":      status,
		"response":    responsePayload,
		"error":       req.Error,
		"finished_at": time.Now().UnixMilli(),
	}
	if err := global.NAV_DB.Model(&domains.AgentTask{}).Where("guid = ?", taskGuid).Updates(updates).Error; err != nil {
		return nil, err
	}
	var task domains.AgentTask
	if err := global.NAV_DB.Where("guid = ?", taskGuid).First(&task).Error; err != nil {
		return nil, err
	}
	agentWaiters.complete(task)
	return &task, nil
}

func (s AgentService) Dispatch(nodeGuid, taskType string, request any, timeout time.Duration, response any) (*domains.AgentTask, error) {
	if nodeGuid == "" {
		return nil, errors.New("missing node guid")
	}
	if taskType == "" {
		return nil, errors.New("missing agent task type")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	guid := strings.ReplaceAll(uuid.NewString(), "-", "")
	ch := agentWaiters.register(guid)
	defer agentWaiters.unregister(guid)
	task := domains.AgentTask{
		BaseDataEntity: commonDomains.BaseDataEntity{Guid: guid},
		NodeGuid:       nodeGuid,
		TaskType:       taskType,
		Status:         domains.AgentTaskStatusPending,
		Request:        string(payload),
		TimeoutMs:      timeout.Milliseconds(),
	}
	if err := s.taskCrud.Create(task); err != nil {
		return nil, err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case completed := <-ch:
			return decodeAgentTaskResponse(completed, response)
		case <-ticker.C:
			completed, done := s.completedTask(guid)
			if done {
				return decodeAgentTaskResponse(completed, response)
			}
		case <-timer.C:
			_ = global.NAV_DB.Model(&domains.AgentTask{}).Where("guid = ? AND status IN ?", guid, []string{domains.AgentTaskStatusPending, domains.AgentTaskStatusDispatched}).Updates(map[string]any{
				"status":      domains.AgentTaskStatusTimeout,
				"error":       "agent task timeout",
				"finished_at": time.Now().UnixMilli(),
			}).Error
			return &task, errors.New("agent task timeout")
		}
	}
}

func (s AgentService) completedTask(guid string) (domains.AgentTask, bool) {
	var task domains.AgentTask
	result := global.NAV_DB.Where("guid = ?", guid).First(&task)
	if result.Error != nil || result.RowsAffected == 0 {
		return domains.AgentTask{}, false
	}
	switch task.Status {
	case domains.AgentTaskStatusSuccess, domains.AgentTaskStatusFailed, domains.AgentTaskStatusTimeout:
		return task, true
	default:
		return domains.AgentTask{}, false
	}
}

func (s AgentService) resolveNode(nodeGuid, agentID string) (*domains.Node, error) {
	if nodeGuid != "" {
		return ServiceGroupApp.NodeService.Get(nodeGuid)
	}
	if agentID == "" {
		return nil, errors.New("missing node guid or agent id")
	}
	var node domains.Node
	result := global.NAV_DB.Where("agent_id = ?", agentID).First(&node)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, errors.New("agent node not found")
	}
	return &node, nil
}

func decodeAgentTaskResponse(task domains.AgentTask, response any) (*domains.AgentTask, error) {
	if task.Status != domains.AgentTaskStatusSuccess {
		if task.Error != "" {
			return &task, errors.New(task.Error)
		}
		return &task, fmt.Errorf("agent task %s", task.Status)
	}
	if response != nil && task.Response != "" {
		if err := json.Unmarshal([]byte(task.Response), response); err != nil {
			return &task, err
		}
	}
	return &task, nil
}

func (h *agentWaiterHub) register(guid string) chan domains.AgentTask {
	ch := make(chan domains.AgentTask, 1)
	h.mu.Lock()
	h.waiters[guid] = ch
	h.mu.Unlock()
	return ch
}

func (h *agentWaiterHub) unregister(guid string) {
	h.mu.Lock()
	delete(h.waiters, guid)
	h.mu.Unlock()
}

func (h *agentWaiterHub) complete(task domains.AgentTask) {
	h.mu.Lock()
	ch := h.waiters[task.Guid]
	h.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- task:
	default:
	}
}

func fallback(value, fallbackValue string) string {
	if value != "" {
		return value
	}
	return fallbackValue
}

func logAgentError(message string, err error) {
	if global.NAV_LOG != nil {
		global.NAV_LOG.Warn(message, zap.Error(err))
	}
}
