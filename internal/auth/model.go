// Package auth implements user authentication and registration.
package auth

import "gorm.io/gorm"

// User represents a registered user in the system.
type User struct {
	gorm.Model
	Name         string `gorm:"not null"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Score        int    `gorm:"default:0;not null"`
}
