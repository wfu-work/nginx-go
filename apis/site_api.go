package apis

import (
	"nginx-go/domains"

	"github.com/gin-gonic/gin"
	commonDomains "github.com/wfu-work/nav-common-go-lib/domains"
	"github.com/wfu-work/nav-common-go-lib/response"
	commonUtils "github.com/wfu-work/nav-common-go-lib/utils"
)

type SiteApi struct{}

// List returns paginated sites.
func (SiteApi) List(c *gin.Context) {
	params := queryParams(c)
	items, total, err := siteService.List(params)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(commonDomains.PageResult{Data: items, Total: total, Page: commonUtils.Str2Int(params["page"]), Size: commonUtils.Str2Int(params["size"])}, c)
}

// Create stores a site.
func (SiteApi) Create(c *gin.Context) {
	var site domains.Site
	if err := c.ShouldBindJSON(&site); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := siteService.Create(site); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// Get returns a site with its location rules.
func (SiteApi) Get(c *gin.Context) {
	result, err := siteService.Get(c.Param("guid"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(result, c)
}

// Update modifies a site.
func (SiteApi) Update(c *gin.Context) {
	var site domains.Site
	if err := c.ShouldBindJSON(&site); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := siteService.Update(c.Param("guid"), site); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// Delete removes a site by guid.
func (SiteApi) Delete(c *gin.Context) {
	if err := siteService.Delete(c.Param("guid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// Enable marks a site as enabled for config rendering.
func (SiteApi) Enable(c *gin.Context) {
	if err := siteService.Enabled(c.Param("guid"), true); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// Disable marks a site as disabled for config rendering.
func (SiteApi) Disable(c *gin.Context) {
	if err := siteService.Enabled(c.Param("guid"), false); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// CreateLocation stores a location rule under a site.
func (SiteApi) CreateLocation(c *gin.Context) {
	var location domains.LocationRule
	if err := c.ShouldBindJSON(&location); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if location.SiteGuid == "" {
		location.SiteGuid = c.Param("guid")
	}
	if err := siteService.CreateLocation(location); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// UpdateLocation modifies one location rule.
func (SiteApi) UpdateLocation(c *gin.Context) {
	var location domains.LocationRule
	if err := c.ShouldBindJSON(&location); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := siteService.UpdateLocation(c.Param("locationGuid"), location); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}

// DeleteLocation removes one location rule.
func (SiteApi) DeleteLocation(c *gin.Context) {
	if err := siteService.DeleteLocation(c.Param("locationGuid")); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.Ok(true, c)
}
