package auth

import "errors"

// ErrEmailTaken is returned when a registration is attempted with an email that already exists.
var ErrEmailTaken = errors.New("auth: email already taken")
