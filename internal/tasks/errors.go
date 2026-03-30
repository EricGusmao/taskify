package tasks

import "errors"

// ErrTeamNotFound is returned when the requested team does not exist.
var ErrTeamNotFound = errors.New("team not found")
