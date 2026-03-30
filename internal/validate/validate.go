// Package validate provides a shared validator for request payload validation.
package validate

import "github.com/go-playground/validator/v10"

var v = validator.New(validator.WithRequiredStructEnabled())

// Struct validates a struct using the package-level validator.
func Struct(s any) error {
	return v.Struct(s)
}
