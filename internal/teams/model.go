package teams

import (
	"time"

	"github.com/EricGusmao/taskify/internal/auth"
	"gorm.io/gorm"
)

// Team represents a group of users working on tasks.
type Team struct {
	gorm.Model
	Name  string      `gorm:"uniqueIndex;not null;size:255"`
	Users []auth.User `gorm:"many2many:members;"`
}

// Member is the join table for the Team–User many-to-many relationship.
type Member struct {
	TeamID    uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"primaryKey"`
	CreatedAt time.Time
}
