package validators

import (
	"net/http"
	"reflect"
	"zori/services/analytics/filters"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
)

type FiltersValidator struct {
	validator *validator.Validate
}

func (f *FiltersValidator) Validate(request any) error {
	if err := f.validator.Struct(request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if reflect.TypeOf(request) == reflect.TypeOf(filters.SectionFilter{}) {
		filters := request.(filters.SectionFilter)
		if err := filters.ValidateTimeRange(); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	}

	return nil
}

func NewFiltersValidator() *FiltersValidator {
	return &FiltersValidator{
		validator: validator.New(),
	}
}
