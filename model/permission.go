package model

import (
	"time"

	"gorm.io/gorm"
)

// Permission 权限点：菜单 1、按钮 2、接口 3
type Permission struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID   *uint64        `gorm:"index" json:"parentId"`
	PermName   string         `gorm:"size:64;not null" json:"permName"`
	PermCode   string         `gorm:"size:100;uniqueIndex;not null" json:"permCode"`
	PermType   int8           `gorm:"not null;default:1;index" json:"permType"`
	RoutePath  string         `gorm:"size:200" json:"routePath"`
	APIPath    string         `gorm:"column:api_path;size:200" json:"apiPath"`
	HTTPMethod string         `gorm:"size:10" json:"httpMethod"`
	Icon       string         `gorm:"size:64" json:"icon"`
	SortNo     int            `gorm:"not null;default:0" json:"sortNo"`
	Status     int8           `gorm:"not null;default:1" json:"status"`
	Remark     string         `gorm:"size:255" json:"remark"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Permission) TableName() string { return "permission" }
