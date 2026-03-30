package tasks

import "errors"

// ErrTeamNotFound is returned when the requested team does not exist.
var ErrTeamNotFound = errors.New("team not found")

// ErrTaskNotFound is returned when the requested task does not exist.
var ErrTaskNotFound = errors.New("task not found")

// ErrAlreadyCompleted is returned when trying to complete an already-completed task.
var ErrAlreadyCompleted = errors.New("task already completed")

// ErrNotMember is returned when the user is not a member of the task's team.
var ErrNotMember = errors.New("user is not a member of this team")
