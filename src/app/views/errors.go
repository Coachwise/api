package views

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// AbortValidation reports a request-binding failure as CodeValidation, including
// the offending json field and rule so the client can localize per field.
func AbortValidation(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"code":  int(CodeValidation),
		"error": bindErrorMessage(err),
		"field": bindErrorField(err),
		"rule":  bindErrorRule(err),
	})
}

// Report validation failures by the request's json field name (e.g. "last_name")
// instead of the Go struct field ("LastName").
func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})
	}
}

// bindErrorMessage turns a request-binding error (field validation failures or
// malformed JSON) into a single, human-readable message safe to return to the
// client — instead of leaking the raw go-playground/validator string.
func bindErrorMessage(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		fe := ve[0]
		field := humanizeField(fe.Field())
		switch fe.Tag() {
		case "required":
			return fmt.Sprintf("%s is required", field)
		case "email":
			return fmt.Sprintf("%s must be a valid email address", field)
		case "min":
			return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
		case "max":
			return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
		case "e164":
			return fmt.Sprintf("%s must be a valid phone number", field)
		default:
			return fmt.Sprintf("%s is invalid", field)
		}
	}
	return "Invalid request"
}

// bindErrorField returns the json field name of the first validation failure.
func bindErrorField(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		return ve[0].Field()
	}
	return ""
}

// bindErrorRule returns the failed validation tag (e.g. "required", "min").
func bindErrorRule(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		return ve[0].Tag()
	}
	return ""
}

func humanizeField(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	if name == "" {
		return "This field"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
