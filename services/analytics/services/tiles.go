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
	timelineTile             *tiles.TimelineTile
	trafficRefererSourceTile *tiles.TrafficRefererSourceTile
	trafficCountrySourceTile *tiles.TrafficCountrySourceTile
	trafficUTMSourceTile     *tiles.TrafficUTMSourceTile
	uniqueVisitorsTile       *tiles.UniqueVisitorsTile
	uniqueSessionsTile       *tiles.UniqueSessionsTile
	bounceRateTile           *tiles.BounceRateTile
	sessionDurationTile      *tiles.SessionDurationTile
	pagesPerSessionTile      *tiles.PagesPerSessionTile
	dauTile                  *tiles.DAUTile
	wauTile                  *tiles.WAUTile
	mauTile                  *tiles.MAUTile
	returnRateTile           *tiles.ReturnRateTile
	timeBetweenVisitsTile    *tiles.TimeBetweenVisitsTile
	visitorsByBrowserTile    *tiles.VisitorsByBrowserTile
	visitorsByOSTile         *tiles.VisitorsByOSTile
	entryPagesTile           *tiles.EntryPagesTile
	exitPagesTile            *tiles.ExitPagesTile
}

func NewTilesService(
	timelineTile *tiles.TimelineTile,
	trafficRefererSourceTile *tiles.TrafficRefererSourceTile,
	trafficCountrySourceTile *tiles.TrafficCountrySourceTile,
	trafficUTMSourceTile *tiles.TrafficUTMSourceTile,
	uniqueVisitorsTile *tiles.UniqueVisitorsTile,
	uniqueSessionsTile *tiles.UniqueSessionsTile,
	bounceRateTile *tiles.BounceRateTile,
	sessionDurationTile *tiles.SessionDurationTile,
	pagesPerSessionTile *tiles.PagesPerSessionTile,
	dauTile *tiles.DAUTile,
	wauTile *tiles.WAUTile,
	mauTile *tiles.MAUTile,
	returnRateTile *tiles.ReturnRateTile,
	timeBetweenVisitsTile *tiles.TimeBetweenVisitsTile,
	visitorsByBrowserTile *tiles.VisitorsByBrowserTile,
	visitorsByOSTile *tiles.VisitorsByOSTile,
	entryPagesTile *tiles.EntryPagesTile,
	exitPagesTile *tiles.ExitPagesTile,
) *TilesService {
	return &TilesService{
		timelineTile:             timelineTile,
		trafficRefererSourceTile: trafficRefererSourceTile,
		trafficCountrySourceTile: trafficCountrySourceTile,
		trafficUTMSourceTile:     trafficUTMSourceTile,
		uniqueVisitorsTile:       uniqueVisitorsTile,
		uniqueSessionsTile:       uniqueSessionsTile,
		bounceRateTile:           bounceRateTile,
		sessionDurationTile:      sessionDurationTile,
		pagesPerSessionTile:      pagesPerSessionTile,
		dauTile:                  dauTile,
		wauTile:                  wauTile,
		mauTile:                  mauTile,
		returnRateTile:           returnRateTile,
		timeBetweenVisitsTile:    timeBetweenVisitsTile,
		visitorsByOSTile:         visitorsByOSTile,
		visitorsByBrowserTile:    visitorsByBrowserTile,
		entryPagesTile:           entryPagesTile,
		exitPagesTile:            exitPagesTile,
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

// GetTrafficSourceRefererTile returns unique visitor counts grouped by referer
// @Summary Get traffic by referer
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
// @Router /api/v1/analytics/tiles/traffic-by-referer [get]
func (s *TilesService) GetTrafficSourceRefererTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.RefererTrafficSourceResponse, error) {
	return s.trafficRefererSourceTile.FetchByReferer(ctx, filter)
}

// GetTrafficSourceCountriesTile returns unique visitor counts grouped by country
// @Summary Get traffic by country
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
// @Router /api/v1/analytics/tiles/traffic-by-country [get]
func (s *TilesService) GetTrafficSourceCountriesTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.CountryTrafficSourceResponse, error) {
	return s.trafficCountrySourceTile.FetchByCountry(ctx, filter)
}

// GetTrafficSourceUTMTile returns unique visitor counts grouped by UTM parameters
// @Summary Get traffic by UTM parameters
// @Description Get unique visitor counts grouped by UTM source, medium, and campaign
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.UTMTrafficSourceResponse "Unique visitors grouped by UTM parameters"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/traffic-by-utm [get]
func (s *TilesService) GetTrafficSourceUTMTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.UTMTrafficSourceResponse, error) {
	return s.trafficUTMSourceTile.FetchByUTM(ctx, filter)
}

// GetUniqueVisitorsTile returns the count of unique visitors for current and previous periods
// @Summary Get unique visitors tile
// @Description Get unique visitor count for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.UniqueVisitorsResponse "Unique visitors count with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/unique-visitors [get]
func (s *TilesService) GetUniqueVisitorsTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.UniqueVisitorsResponse, error) {
	return s.uniqueVisitorsTile.Fetch(ctx, filter)
}

// GetUniqueSessionsTile returns the count of unique sessions for current and previous periods
// @Summary Get unique sessions tile
// @Description Get unique session count for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.UniqueSessionsResponse "Unique sessions count with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/unique-sessions [get]
func (s *TilesService) GetUniqueSessionsTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.UniqueSessionsResponse, error) {
	return s.uniqueSessionsTile.Fetch(ctx, filter)
}

// GetBounceRateTile returns the bounce rate for current and previous periods
// @Summary Get bounce rate tile
// @Description Get bounce rate percentage for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.BounceRateResponse "Bounce rate with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/bounce-rate [get]
func (s *TilesService) GetBounceRateTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.BounceRateResponse, error) {
	return s.bounceRateTile.Fetch(ctx, filter)
}

// GetSessionDurationTile returns the average session duration for current and previous periods
// @Summary Get session duration tile
// @Description Get average session duration in seconds for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.SessionDurationResponse "Session duration with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/session-duration [get]
func (s *TilesService) GetSessionDurationTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.SessionDurationResponse, error) {
	return s.sessionDurationTile.Fetch(ctx, filter)
}

// GetPagesPerSessionTile returns the average pages per session for current and previous periods
// @Summary Get pages per session tile
// @Description Get average number of pages viewed per session for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.PagesPerSessionResponse "Pages per session with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/pages-per-session [get]
func (s *TilesService) GetPagesPerSessionTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.PagesPerSessionResponse, error) {
	return s.pagesPerSessionTile.Fetch(ctx, filter)
}

// GetDAUTile returns daily active users for current and previous periods
// @Summary Get daily active users tile
// @Description Get daily active user count (last 24h) compared to the previous day
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.DAUResponse "Daily active users with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/dau [get]
func (s *TilesService) GetDAUTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.DAUResponse, error) {
	return s.dauTile.Fetch(ctx, filter)
}

// GetWAUTile returns weekly active users for current and previous periods
// @Summary Get weekly active users tile
// @Description Get weekly active user count (last 7 days) compared to the previous week
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.WAUResponse "Weekly active users with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/wau [get]
func (s *TilesService) GetWAUTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.WAUResponse, error) {
	return s.wauTile.Fetch(ctx, filter)
}

// GetMAUTile returns monthly active users for current and previous periods
// @Summary Get monthly active users tile
// @Description Get monthly active user count (last 30 days) compared to the previous month
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.MAUResponse "Monthly active users with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/mau [get]
func (s *TilesService) GetMAUTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.MAUResponse, error) {
	return s.mauTile.Fetch(ctx, filter)
}

// GetReturnRateTile returns the return rate for current and previous periods
// @Summary Get return rate tile
// @Description Get percentage of visitors with more than one session for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.ReturnRateResponse "Return rate with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/return-rate [get]
func (s *TilesService) GetReturnRateTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.ReturnRateResponse, error) {
	return s.returnRateTile.Fetch(ctx, filter)
}

// GetTimeBetweenVisitsTile returns the average time between visits for current and previous periods
// @Summary Get time between visits tile
// @Description Get average hours between consecutive visits per visitor for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.TimeBetweenVisitsResponse "Time between visits with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/time-between-visits [get]
func (s *TilesService) GetTimeBetweenVisitsTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.TimeBetweenVisitsResponse, error) {
	return s.timeBetweenVisitsTile.Fetch(ctx, filter)
}

// GetVisitorsByBrowserTile returns the number of visitors by browser for current and previous periods
// @Summary Get visitors by browser tile
// @Description Get number of visitors by browser for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.VisitorsByBrowserResponse "Visitors by browser with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/visitors-by-browser [get]
func (s *TilesService) GetVisitorsByBrowserTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.VisitorsByBrowserResponse, error) {
	return s.visitorsByBrowserTile.Fetch(ctx, filter)
}

// GetVisitorsByOSTile returns the number of visitors by OS for current and previous periods
// @Summary Get visitors by OS tile
// @Description Get number of visitors by OS for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.VisitorsByOSResponse "Visitors by OS with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/visitors-by-os [get]
func (s *TilesService) GetVisitorsByOSTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.VisitorsByOSResponse, error) {
	return s.visitorsByOSTile.Fetch(ctx, filter)
}

// GetEntryPagesTile returns the top entry pages where visitors first enter the site
// @Summary Get entry pages tile
// @Description Get top entry pages where new visitors first land for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.EntryPagesResponse "Entry pages with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/entry-pages [get]
func (s *TilesService) GetEntryPagesTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.EntryPagesResponse, error) {
	return s.entryPagesTile.Fetch(ctx, filter)
}

// GetExitPagesTile returns the top exit pages where sessions end
// @Summary Get exit pages tile
// @Description Get top exit pages where sessions end for current period compared to the previous period
// @Tags Analytics
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param filter query filters.SectionFilter true "Filter parameters"
// @Success 200 {object} tiles.ExitPagesResponse "Exit pages with period comparison"
// @Failure 400 {object} map[string]interface{} "Invalid request parameters"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Invalid or missing JWT token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/analytics/tiles/exit-pages [get]
func (s *TilesService) GetExitPagesTile(ctx *ctx.Ctx, filter *filters.SectionFilter) (*tiles.ExitPagesResponse, error) {
	return s.exitPagesTile.Fetch(ctx, filter)
}

