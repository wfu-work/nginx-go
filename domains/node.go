package domains

import commonDomains "github.com/wfu-work/nav-common-go-lib/domains"

const (
	NodeAccessLocal = "local"
	NodeAccessAgent = "agent"

	NodeStatusOnline  = "online"
	NodeStatusOffline = "offline"

	AgentTaskStatusPending    = "pending"
	AgentTaskStatusDispatched = "dispatched"
	AgentTaskStatusSuccess    = "success"
	AgentTaskStatusFailed     = "failed"
	AgentTaskStatusTimeout    = "timeout"
)

// Node describes one server that can host one or more nginx instances.
type Node struct {
	commonDomains.BaseDataEntity
	Name        string `gorm:"column:name;size:120;index" json:"name"`
	AccessMode  string `gorm:"column:access_mode;size:30;index" json:"accessMode"`
	AgentID     string `gorm:"column:agent_id;size:120;uniqueIndex" json:"agentId"`
	Address     string `gorm:"column:address;size:255" json:"address"`
	Labels      string `gorm:"column:labels;size:500" json:"labels"`
	Status      string `gorm:"column:status;size:30;index" json:"status"`
	Version     string `gorm:"column:version;size:80" json:"version"`
	LastSeenAt  int64  `gorm:"column:last_seen_at;index" json:"lastSeenAt"`
	Enabled     bool   `gorm:"column:enabled;index" json:"enabled"`
	Description string `gorm:"column:description;size:500" json:"description"`
}

func (Node) TableName() string {
	return "nginx_nodes"
}

// AgentTask is the center-side task queue consumed by nginx agents.
type AgentTask struct {
	commonDomains.BaseDataEntity
	NodeGuid   string `gorm:"column:node_guid;size:50;index" json:"nodeGuid"`
	TaskType   string `gorm:"column:task_type;size:80;index" json:"taskType"`
	Status     string `gorm:"column:status;size:30;index" json:"status"`
	Request    string `gorm:"column:request;type:text" json:"request"`
	Response   string `gorm:"column:response;type:text" json:"response"`
	Error      string `gorm:"column:error;size:1000" json:"error"`
	TimeoutMs  int64  `gorm:"column:timeout_ms" json:"timeoutMs"`
	Dispatched int64  `gorm:"column:dispatched_at;index" json:"dispatchedAt"`
	FinishedAt int64  `gorm:"column:finished_at;index" json:"finishedAt"`
}

func (AgentTask) TableName() string {
	return "nginx_agent_tasks"
}
