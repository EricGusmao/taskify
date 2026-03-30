package teams

import "errors"

var (
	// ErrNameTaken is returned when a team name is already in use.
	ErrNameTaken = errors.New("team name is already in use")
	// ErrTeamNotFound is returned when the requested team does not exist.
	ErrTeamNotFound = errors.New("team not found")
	// ErrUserNotFound is returned when the requested user does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrAlreadyMember is returned when the user is already a member of the team.
	ErrAlreadyMember = errors.New("user is already a member of the team")
)
