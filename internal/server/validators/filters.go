package validators

import (
	"fmt"
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

	if value, err := getField(request, "TimeBoundaries"); err == nil {
		if tr, err := filters.ValidateTimeRange(value.(filters.TimeBoundaries)); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		} else {
			if err := setField(request, "TimeRange", tr); err != nil {
				fmt.Println("Err ", err)
			}
		}
	}

	return nil
}

func NewFiltersValidator() *FiltersValidator {
	return &FiltersValidator{
		validator: validator.New(),
	}
}

func getField(obj interface{}, fieldName string) (interface{}, error) {
	rv := reflect.ValueOf(obj)

	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct")
	}

	fieldVal := rv.FieldByName(fieldName)
	if !fieldVal.IsValid() {
		return nil, fmt.Errorf("field %s not found", fieldName)
	}

	return fieldVal.Interface(), nil
}

func setField(obj interface{}, fieldName string, value interface{}) error {
	rv := reflect.ValueOf(obj)

	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to struct")
	}

	rv = rv.Elem()

	fieldVal := rv.FieldByName(fieldName)
	if !fieldVal.IsValid() {
		return fmt.Errorf("field %s not found", fieldName)
	}

	if !fieldVal.CanSet() {
		return fmt.Errorf("field %s cannot be set", fieldName)
	}

	val := reflect.ValueOf(value)

	if fieldVal.Type() != val.Type() {
		return fmt.Errorf("value type %v doesn't match field type %v",
			val.Type(), fieldVal.Type())
	}

	fieldVal.Set(val)
	return nil
}
