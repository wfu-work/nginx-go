package services

import (
	"errors"
	"nginx-go/domains"

	"github.com/wfu-work/nav-common-go-lib/global"
	commonServices "github.com/wfu-work/nav-common-go-lib/services"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type SiteService struct {
	commonServices.CrudService[domains.Site]
	locationCrud commonServices.CrudService[domains.LocationRule]
}

func (s SiteService) List(params map[string]string) (interface{}, int64, error) {
	return s.CrudService.List(commonUtils.ToPageInfo(params), "name,serverName")
}

func (s SiteService) Create(site domains.Site) error {
	defaultSite(&site)
	return s.CrudService.Create(site)
}

func (s SiteService) Update(guid string, site domains.Site) error {
	if guid == "" {
		return errors.New("missing site guid")
	}
	site.Guid = guid
	defaultSite(&site)
	return s.CrudService.Updates(site)
}

func (s SiteService) Delete(guid string) error {
	if guid == "" {
		return errors.New("missing site guid")
	}
	return s.CrudService.DeleteByGuid(guid)
}

func (s SiteService) Get(guid string) (map[string]any, error) {
	site, err := s.CrudService.GetByGuid(guid)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, errors.New("site not found")
	}
	locations, err := s.locationCrud.ListByFields(map[string]any{"siteGuid": guid})
	if err != nil {
		return nil, err
	}
	return map[string]any{"site": site, "locations": locations}, nil
}

func (s SiteService) LocationList(siteGuid string) ([]domains.LocationRule, error) {
	if siteGuid == "" {
		return nil, errors.New("missing site guid")
	}
	return s.locationCrud.ListByFields(map[string]any{"siteGuid": siteGuid})
}

func (s SiteService) CreateLocation(location domains.LocationRule) error {
	defaultLocation(&location)
	return s.locationCrud.Create(location)
}

func (s SiteService) UpdateLocation(guid string, location domains.LocationRule) error {
	if guid == "" {
		return errors.New("missing location guid")
	}
	location.Guid = guid
	defaultLocation(&location)
	return s.locationCrud.Updates(location)
}

func (s SiteService) DeleteLocation(guid string) error {
	if guid == "" {
		return errors.New("missing location guid")
	}
	return s.locationCrud.DeleteByGuid(guid)
}

func (s SiteService) Enabled(guid string, enabled bool) error {
	site, err := s.CrudService.GetByGuid(guid)
	if err != nil {
		return err
	}
	if site == nil {
		return errors.New("site not found")
	}
	site.Enabled = enabled
	return global.NAV_DB.Model(&domains.Site{}).Where("guid = ?", guid).Update("enabled", enabled).Error
}

func defaultSite(site *domains.Site) {
	if site.Listen == "" {
		site.Listen = "80"
	}
	if site.Index == "" {
		site.Index = "index.html index.htm"
	}
}

func defaultLocation(location *domains.LocationRule) {
	if location.Path == "" {
		location.Path = "/"
	}
}
