package model

import "time"

// SystemUser is the database model for the system_user table.
type SystemUser struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"column:username;size:64;not null;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"column:password_hash;size:255;not null" json:"-"`
	Nickname     string    `gorm:"column:nickname;size:100;not null" json:"nickname"`
	Status       int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (SystemUser) TableName() string {
	return "system_user"
}
