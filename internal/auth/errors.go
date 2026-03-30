package auth

import "errors"

// ErrEmailTaken is returned when a registration is attempted with an email that already exists.
var ErrEmailTaken = errors.New("auth: email already taken")

// ErrInvalidCredentials is returned when login credentials are invalid.
// Used for both unknown email and wrong password to prevent user enumeration.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")
