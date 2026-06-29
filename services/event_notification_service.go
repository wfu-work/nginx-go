package services

import (
	"encoding/json"
	"errors"
	"nginx-go/domains"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type EventNotificationService struct {
	commonServices.CrudService[domains.EventNotification]
}

type EventNotificationCreate struct {
	Title      string
	Content    string
	Level      string
	SourceType string
	SourceGuid string
	EventTime  int64
}

type websocketMessage struct {
	Type string `json:"type"`
	Data any    `json:"data"`
	Time int64  `json:"time"`
}

var notificationHub = newNotificationBroker()

// List returns paginated notifications with exact filters for read, level, and source.
func (s EventNotificationService) List(params map[string]string) (interface{}, int64, error) {
	if global.NAV_DB == nil {
		return nil, 0, errors.New("database is not initialized")
	}
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Page <= 0 {
		pageInfo.Page = 1
	}
	if pageInfo.Size <= 0 {
		pageInfo.Size = 20
	}
	db := global.NAV_DB.Model(&domains.EventNotification{})
	if read := params["read"]; read != "" {
		db = db.Where("is_read = ?", read == "1" || strings.EqualFold(read, "true"))
	}
	if level := params["level"]; level != "" {
		db = db.Where("level = ?", level)
	}
	if sourceType := params["sourceType"]; sourceType != "" {
		db = db.Where("source_type = ?", sourceType)
	}
	if sourceGuid := params["sourceGuid"]; sourceGuid != "" {
		db = db.Where("source_guid = ?", sourceGuid)
	}
	if keyword := params["keyword"]; keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("title like ? OR content like ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []domains.EventNotification
	err := db.Order("event_time desc, id desc").Limit(pageInfo.Size).Offset(pageInfo.Size * (pageInfo.Page - 1)).Find(&rows).Error
	return rows, total, err
}

// CreateNotification stores a notification and broadcasts it to connected clients.
func (s EventNotificationService) CreateNotification(req EventNotificationCreate) (*domains.EventNotification, error) {
	if req.Title == "" {
		req.Title = "系统通知"
	}
	if req.Level == "" {
		req.Level = domains.EventNotificationLevelInfo
	}
	if req.EventTime == 0 {
		req.EventTime = time.Now().UnixMilli()
	}
	if existing := s.recentDuplicate(req); existing != nil {
		return existing, nil
	}
	item := domains.EventNotification{
		BaseDataEntity: commonDomains.BaseDataEntity{Guid: strings.ReplaceAll(uuid.NewString(), "-", "")},
		Title:          req.Title,
		Content:        req.Content,
		Level:          req.Level,
		Read:           false,
		SourceType:     req.SourceType,
		SourceGuid:     req.SourceGuid,
		EventTime:      req.EventTime,
	}
	if err := s.Create(item); err != nil {
		return nil, err
	}
	notificationHub.Broadcast("notification.created", item)
	return &item, nil
}

func (s EventNotificationService) recentDuplicate(req EventNotificationCreate) *domains.EventNotification {
	if req.SourceType == "" || req.SourceGuid == "" || req.Title == "" {
		return nil
	}
	var existing domains.EventNotification
	result := global.NAV_DB.
		Where("source_type = ? AND source_guid = ? AND title = ? AND is_read = ? AND event_time >= ?", req.SourceType, req.SourceGuid, req.Title, false, req.EventTime-5*60*1000).
		Order("event_time desc, id desc").
		First(&existing)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil
	}
	return &existing
}

// Notify is a best-effort helper for business services.
func (s EventNotificationService) Notify(req EventNotificationCreate) {
	if global.NAV_DB == nil {
		return
	}
	if _, err := s.CreateNotification(req); err != nil && global.NAV_LOG != nil {
		global.NAV_LOG.Warn("create event notification failed", zap.Error(err))
	}
}

func (s EventNotificationService) MarkRead(guid string) error {
	if guid == "" {
		return errors.New("missing notification guid")
	}
	return global.NAV_DB.Model(&domains.EventNotification{}).Where("guid = ?", guid).Update("is_read", true).Error
}

func (s EventNotificationService) MarkAllRead() error {
	return global.NAV_DB.Model(&domains.EventNotification{}).Where("is_read = ?", false).Update("is_read", true).Error
}

func (EventNotificationService) Subscribe() (<-chan []byte, func()) {
	return notificationHub.Subscribe()
}

type notificationBroker struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func newNotificationBroker() *notificationBroker {
	return &notificationBroker{clients: make(map[chan []byte]struct{})}
}

func (b *notificationBroker) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	cancel := func() {
		b.mu.Lock()
		if _, ok := b.clients[ch]; ok {
			delete(b.clients, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
	return ch, cancel
}

func (b *notificationBroker) Broadcast(eventType string, data any) {
	payload, err := json.Marshal(websocketMessage{Type: eventType, Data: data, Time: time.Now().UnixMilli()})
	if err != nil {
		return
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- payload:
		default:
		}
	}
}
