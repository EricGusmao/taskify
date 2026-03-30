// Package users implements the user profile management slice.
package users

import "errors"

// ErrUnsupportedFormat is returned when the uploaded file is not a supported image type.
var ErrUnsupportedFormat = errors.New("unsupported file format")

// ErrImageTooLarge is returned when the image dimensions exceed the allowed pixel budget.
var ErrImageTooLarge = errors.New("image dimensions too large")

// ErrUserNotFound is returned when the target user does not exist.
var ErrUserNotFound = errors.New("user not found")
