package tasks

import (
	"time"

	"gorm.io/gorm"
)

// Task represents a task belonging to a team.
type Task struct {
	gorm.Model
	Title        string     `gorm:"not null;size:255"`
	Points       int        `gorm:"not null"`
	TeamID       uint       `gorm:"not null;index"`
	DoneByUserID *uint      `gorm:"index"`
	DoneAt       *time.Time
}
