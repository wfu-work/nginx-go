package services

import (
	"encoding/json"
	"nginx-go/domains"

	"github.com/google/uuid"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/global"
	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
	"go.uber.org/zap"
)

type AuditService struct {
	commonServices.CrudService[domains.AuditLog]
}

type AuditRecord struct {
	Action       string
	ResourceType string
	ResourceGuid string
	Success      bool
	Message      string
	Reason       string
	Detail       any
}

// Record writes a best-effort audit log; audit failures are logged but do not break user actions.
func (s AuditService) Record(record AuditRecord) {
	if global.NAV_DB == nil {
		return
	}
	detail := ""
	if record.Detail != nil {
		if payload, err := json.Marshal(record.Detail); err == nil {
			detail = string(payload)
		}
	}
	log := domains.AuditLog{
		BaseDataEntity: commonDomains.BaseDataEntity{Guid: uuid.NewString()},
		Action:         record.Action,
		ResourceType:   record.ResourceType,
		ResourceGuid:   record.ResourceGuid,
		Success:        record.Success,
		Status:         statusText(record.Success),
		Message:        record.Message,
		Reason:         record.Reason,
		Detail:         detail,
	}
	if err := s.Create(log); err != nil && global.NAV_LOG != nil {
		global.NAV_LOG.Warn("record nginx audit log failed", zap.Error(err))
	}
}

// List returns paginated audit records.
func (s AuditService) List(params map[string]string) (interface{}, int64, error) {
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Desc == "" && pageInfo.Asc == "" {
		pageInfo.Desc = "createTime"
	}
	return s.CrudService.List(pageInfo, "action,resourceType,status,message,reason")
}
