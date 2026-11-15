package model

import "time"

// UserRole 用户角色关联：联合唯一，避免重复授予
type UserRole struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"not null;uniqueIndex:uk_user_role" json:"userId"`
	RoleID    uint64    `gorm:"not null;uniqueIndex:uk_user_role;index" json:"roleId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (UserRole) TableName() string { return "user_role" }

// RolePermission 角色权限关联：授权事务的写入目标
type RolePermission struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID    uint64    `gorm:"not null;uniqueIndex:uk_role_perm" json:"roleId"`
	PermID    uint64    `gorm:"not null;uniqueIndex:uk_role_perm;index:idx_rp_perm" json:"permId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (RolePermission) TableName() string { return "role_permission" }
