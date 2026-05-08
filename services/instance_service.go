package services

import (
	"errors"
	"nginx-go/domains"

	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type InstanceService struct {
	commonServices.CrudService[domains.NginxInstance]
}

func (s InstanceService) List(params map[string]string) (interface{}, int64, error) {
	return s.CrudService.List(commonUtils.ToPageInfo(params), "name,host,mode")
}

func (s InstanceService) Create(instance domains.NginxInstance) error {
	defaultInstance(&instance)
	return s.CrudService.Create(instance)
}

func (s InstanceService) Update(guid string, instance domains.NginxInstance) error {
	if guid == "" {
		return errors.New("missing instance guid")
	}
	instance.Guid = guid
	defaultInstance(&instance)
	return s.CrudService.Updates(instance)
}

func (s InstanceService) Delete(guid string) error {
	if guid == "" {
		return errors.New("missing instance guid")
	}
	return s.CrudService.DeleteByGuid(guid)
}

func (s InstanceService) Get(guid string) (*domains.NginxInstance, error) {
	if guid == "" {
		return nil, errors.New("missing instance guid")
	}
	instance, err := s.GetByGuid(guid)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, errors.New("instance not found")
	}
	return instance, nil
}

func defaultInstance(instance *domains.NginxInstance) {
	if instance.Mode == "" {
		instance.Mode = "command"
	}
	if instance.ServiceName == "" {
		instance.ServiceName = "nginx"
	}
	if instance.Bin == "" {
		instance.Bin = "nginx"
	}
	if instance.Systemctl == "" {
		instance.Systemctl = "systemctl"
	}
}
