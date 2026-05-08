package services

import (
	"errors"
	"nginx-go/domains"

	"github.com/wfu-work/nav-common-go-lib/global"
	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type SettingService struct {
	commonServices.CrudService[domains.Setting]
}

func (s SettingService) List(params map[string]string) (interface{}, int64, error) {
	return s.CrudService.List(commonUtils.ToPageInfo(params), "key,description")
}

func (s SettingService) Save(setting domains.Setting) error {
	if setting.Key == "" {
		return errors.New("missing setting key")
	}
	var existing domains.Setting
	result := global.NAV_DB.Where("key = ?", setting.Key).Find(&existing)
	if result.Error == nil && result.RowsAffected > 0 {
		setting.Guid = existing.Guid
		return s.CrudService.Updates(setting)
	}
	return s.CrudService.Create(setting)
}

func (s SettingService) Delete(guid string) error {
	if guid == "" {
		return errors.New("missing setting guid")
	}
	return s.CrudService.DeleteByGuid(guid)
}
