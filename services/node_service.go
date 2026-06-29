package services

import (
	"errors"
	"nginx-go/domains"
	"time"

	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type NodeService struct {
	commonServices.CrudService[domains.Node]
}

func (s NodeService) List(params map[string]string) (interface{}, int64, error) {
	pageInfo := commonUtils.ToPageInfo(params)
	if pageInfo.Desc == "" && pageInfo.Asc == "" {
		pageInfo.Desc = "lastSeenAt"
	}
	return s.CrudService.List(pageInfo, "name,agentId,address,status,accessMode,labels")
}

func (s NodeService) Create(node domains.Node) error {
	if !node.Enabled {
		node.Enabled = true
	}
	defaultNode(&node)
	return s.CrudService.Create(node)
}

func (s NodeService) Update(guid string, node domains.Node) error {
	if guid == "" {
		return errors.New("missing node guid")
	}
	node.Guid = guid
	defaultNode(&node)
	return s.CrudService.Updates(node)
}

func (s NodeService) Delete(guid string) error {
	if guid == "" {
		return errors.New("missing node guid")
	}
	return s.CrudService.DeleteByGuid(guid)
}

func (s NodeService) Get(guid string) (*domains.Node, error) {
	if guid == "" {
		return nil, errors.New("missing node guid")
	}
	node, err := s.GetByGuid(guid)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, errors.New("node not found")
	}
	return node, nil
}

func defaultNode(node *domains.Node) {
	if node.AccessMode == "" {
		node.AccessMode = domains.NodeAccessAgent
	}
	if node.Status == "" {
		node.Status = domains.NodeStatusOffline
	}
	if node.LastSeenAt == 0 && node.Status == domains.NodeStatusOnline {
		node.LastSeenAt = time.Now().UnixMilli()
	}
}
