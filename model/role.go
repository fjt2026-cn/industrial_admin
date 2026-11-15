package model

import (
	"time"

	"gorm.io/gorm"
)

// Role 角色表：授权管理单元，data_scope 预留数据范围
type Role struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleName  string         `gorm:"size:50;uniqueIndex;not null" json:"roleName"`
	RoleCode  string         `gorm:"size:50;uniqueIndex;not null" json:"roleCode"`
	DataScope int8           `gorm:"not null;default:1" json:"dataScope"`
	IsBuiltin int8           `gorm:"not null;default:0" json:"isBuiltin"`
	SortNo    int            `gorm:"not null;default:0" json:"sortNo"`
	Status    int8           `gorm:"not null;default:1;index" json:"status"`
	Remark    string         `gorm:"size:255" json:"remark"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Role) TableName() string { return "role" }
