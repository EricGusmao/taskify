package teams

import "gorm.io/gorm"

// Team represents a group of users working on tasks.
type Team struct {
	gorm.Model
	Name string `gorm:"uniqueIndex;not null"`
}
