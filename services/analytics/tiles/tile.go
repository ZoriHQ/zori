package tiles

import (
	"zori/internal/ctx"
	"zori/services/analytics/filters"
)

type CardPrecision string

const (
	CardPrecisionMinutes CardPrecision = "minutes"
	CardPrecisionHourly  CardPrecision = "hourly"
	CardPrecisionDaily   CardPrecision = "daily"
	CardPrecisionWeekly  CardPrecision = "weekly"
	CardPrecisionMonthly CardPrecision = "monthly"
)

type AnalyticCard[T any] interface {
	Fetch(ctx *ctx.Ctx, filter filters.SectionFilter) (*T, error)
}

func FilterToTimePrecision(filter *filters.SectionFilter) CardPrecision {
	switch filter.TimeBoundaries {
	case filters.TimeBoundariesHour:
		return CardPrecisionMinutes
	case filters.TimeBoundariesToday:
		return CardPrecisionHourly
	case filters.TimeBoundariesLastWeek:
		return CardPrecisionDaily
	case filters.TimeBoundariesLastMonth:
		return CardPrecisionWeekly
	default:
		return CardPrecisionMonthly
	}
}
