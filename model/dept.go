package model

import (
	"time"

	"gorm.io/gorm"
)

// Dept 部门：parent_id 为空表示根，树形结构由业务层组装
type Dept struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID  *uint64        `gorm:"index" json:"parentId"`
	DeptName  string         `gorm:"size:64;not null" json:"deptName"`
	DeptCode  string         `gorm:"size:64;uniqueIndex;not null" json:"deptCode"`
	LeaderID  *uint64        `gorm:"index" json:"leaderId"`
	SortNo    int            `gorm:"not null;default:0" json:"sortNo"`
	Status    int8           `gorm:"not null;default:1" json:"status"`
	Remark    string         `gorm:"size:255" json:"remark"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Children []*Dept `gorm:"-" json:"children,omitempty"`
}

func (Dept) TableName() string { return "dept" }
