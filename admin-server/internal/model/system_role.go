package model

import "time"

// SystemRole is the database model for the system_role table.
type SystemRole struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;size:64;not null" json:"name"`
	Code      string    `gorm:"column:code;size:64;not null;uniqueIndex" json:"code"`
	Status    int8      `gorm:"column:status;not null;default:1" json:"status"`
	Sort      int       `gorm:"column:sort;not null;default:0" json:"sort"`
	Remark    string    `gorm:"column:remark;size:255;not null;default:''" json:"remark"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (SystemRole) TableName() string {
	return "system_role"
}
