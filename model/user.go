package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户表：口令只存 bcrypt 哈希，字段与库表一一对应
type User struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username       string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash   string         `gorm:"size:100;not null" json:"-"`
	RealName       string         `gorm:"size:50;not null" json:"realName"`
	Email          string         `gorm:"size:100" json:"email"`
	Phone          string         `gorm:"size:20" json:"phone"`
	DeptID         *uint64        `gorm:"index" json:"deptId"`
	Status         int8           `gorm:"not null;default:1;index" json:"status"`
	LoginFailCount int            `gorm:"not null;default:0" json:"-"`
	LockUntil      *time.Time     `json:"-"`
	LastLoginAt    *time.Time     `gorm:"index" json:"lastLoginAt"`
	LastLoginIP    string         `gorm:"column:last_login_ip;size:45" json:"lastLoginIp"`
	Remark         string         `gorm:"size:255" json:"remark"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Dept           *Dept          `gorm:"foreignKey:DeptID" json:"dept,omitempty"`
}

// TableName 表名为保留字，GORM 会自动加反引号
func (User) TableName() string { return "user" }
