package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom tag name function to use json tags for field names
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// ValidateStruct validates a struct using the validator tags
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return NewValidationError(validationErrors)
		}
		return err
	}
	return nil
}

// ValidationError represents validation errors in a more user-friendly format
type ValidationError struct {
	Errors map[string]string `json:"errors"`
}

func (e *ValidationError) Error() string {
	var messages []string
	for field, message := range e.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", field, message))
	}
	return strings.Join(messages, "; ")
}

// NewValidationError creates a new ValidationError from validator.ValidationErrors
func NewValidationError(validationErrors validator.ValidationErrors) *ValidationError {
	errors := make(map[string]string)

	for _, err := range validationErrors {
		fieldName := err.Field()

		switch err.Tag() {
		case "required":
			errors[fieldName] = "This field is required"
		case "email":
			errors[fieldName] = "Must be a valid email address"
		case "min":
			errors[fieldName] = fmt.Sprintf("Must be at least %s characters long", err.Param())
		case "max":
			errors[fieldName] = fmt.Sprintf("Must be no more than %s characters long", err.Param())
		case "alphanum":
			errors[fieldName] = "Must contain only letters and numbers"
		case "oneof":
			errors[fieldName] = fmt.Sprintf("Must be one of: %s", strings.ReplaceAll(err.Param(), " ", ", "))
		default:
			errors[fieldName] = fmt.Sprintf("Validation failed for tag '%s'", err.Tag())
		}
	}

	return &ValidationError{Errors: errors}
}

// IsValidationError checks if the error is a ValidationError
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
