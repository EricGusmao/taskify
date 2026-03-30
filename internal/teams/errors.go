package teams

import "errors"

var (
	// ErrNameTaken is returned when a team name is already in use.
	ErrNameTaken = errors.New("teams.ErrNameTaken")
)
