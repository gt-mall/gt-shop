package model

import (
	"time"

	"gorm.io/gorm"
)

type Member struct {
	ID          int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Nickname    string         `gorm:"column:nickname;size:255;not null;default:''" json:"nickname"`
	AvatarID    *int64         `gorm:"column:avatar_id;default:null" json:"avatar_id"`
	Mobile      *string        `gorm:"column:mobile;size:255;default:null" json:"mobile"`
	Status      int8           `gorm:"column:status;not null" json:"status"`
	LastLoginAt *time.Time     `gorm:"column:last_login_at;default:null" json:"last_login_at"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

func (Member) TableName() string {
	return "member"
}

type MemberAuth struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MemberID     int64     `gorm:"column:member_id;not null;index:idx_member_auth_member_id" json:"member_id"`
	AuthType     string    `gorm:"column:auth_type;size:255;not null;uniqueIndex:uk_auth_type_app_openid,priority:1" json:"auth_type"`
	AppID        string    `gorm:"column:app_id;size:255;not null;uniqueIndex:uk_auth_type_app_openid,priority:2" json:"app_id"`
	OpenID       string    `gorm:"column:openid;size:255;not null;uniqueIndex:uk_auth_type_app_openid,priority:3" json:"openid"`
	UnionID      *string   `gorm:"column:unionid;size:255;default:null" json:"unionid"`
	ProviderData *string   `gorm:"column:provider_data;type:json;default:null" json:"provider_data"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (MemberAuth) TableName() string {
	return "member_auth"
}

type MemberAddress struct {
	ID           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MemberID     int64          `gorm:"column:member_id;not null;index:idx_member_address_member_id" json:"member_id"`
	Name         string         `gorm:"column:name;size:255;not null" json:"name"`
	Mobile       string         `gorm:"column:mobile;size:255;not null" json:"mobile"`
	ProvinceCode string         `gorm:"column:province_code;size:255;not null" json:"province_code"`
	Province     string         `gorm:"column:province;size:255;not null" json:"province"`
	CityCode     string         `gorm:"column:city_code;size:255;not null" json:"city_code"`
	City         string         `gorm:"column:city;size:255;not null" json:"city"`
	DistrictCode string         `gorm:"column:district_code;size:255;not null" json:"district_code"`
	District     string         `gorm:"column:district;size:255;not null" json:"district"`
	Address      string         `gorm:"column:address;size:255;not null" json:"address"`
	PostalCode   *string        `gorm:"column:postal_code;size:255;default:null" json:"postal_code"`
	IsDefault    int8           `gorm:"column:is_default;not null" json:"is_default"`
	CreatedAt    time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

func (MemberAddress) TableName() string {
	return "member_address"
}
