package services

import (
	"zori/internal/ctx"
	"zori/services/analytics/filters"
	"zori/services/analytics/tiles"
)

// TilesService provides methods for managing analytics tiles.
// It contains endpoints definition for various tiles available on the dashboard.
// the tiles code is defined in /services/analytics/tiles/*.tile.view.go files
type TilesService struct {
	timelineTile      *tiles.TimelineTile
	trafficSourceTile *tiles.TrafficSourceTile
}

func NewTilesService(
	timelineTile *tiles.TimelineTile,
	trafficSourceTile *tiles.TrafficSourceTile,
) *TilesService {
	return &TilesService{
		timelineTile:      timelineTile,
		trafficSourceTile: trafficSourceTile,
	}
}

// GetUniqueVisitorsTimeline returns unique visitors over time split by device type
// @Summary Get unique visitors timeline
// @Description Get unique visitor counts over time, split by mobile and desktop devices for chart visualization
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.TimelineTileResponse "Unique visitors timeline data"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/timeline [get]
func (s *TilesService) GetTimelineTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.TimelineTileResponse, error) {
	return s.timelineTile.Fetch(ctx, filter)
}

// GetUniqueVisitorsByOrigin returns unique visitor counts grouped by referer
// @Summary Get unique visitors by traffic origin
// @Description Get unique visitor counts grouped by referrer domain (traffic source)
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.RefererTrafficSourceResponse "Unique visitors grouped by referer"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/origin [get]
func (s *TilesService) GetTrafficSourceRefererTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.RefererTrafficSourceResponse, error) {
	return s.trafficSourceTile.FetchByReferer(ctx, filter)
}

// GetUniqueVisitorsByCountry returns unique visitor counts grouped by country
// @Summary Get unique visitors by country
// @Description Get unique visitor counts grouped by country code
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.CountryTrafficSourceResponse "Unique visitors grouped by country"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/visitors/country [get]
func (s *TilesService) GetTrafficSourceCountriesTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.CountryTrafficSourceResponse, error) {
	return s.trafficSourceTile.FetchByCountry(ctx, filter)
}
