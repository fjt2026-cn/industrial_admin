package model

import (
	"time"

	"gorm.io/gorm"
)

// SysConfig 系统参数：键值对存储，value_type 决定读取时的转换方式
type SysConfig struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigKey   string         `gorm:"size:100;uniqueIndex;not null" json:"configKey"`
	ConfigName  string         `gorm:"size:100;not null" json:"configName"`
	ConfigValue string         `gorm:"size:2000" json:"configValue"`
	ValueType   int8           `gorm:"not null;default:1" json:"valueType"`
	Category    string         `gorm:"size:32;not null;default:base;index" json:"category"`
	IsSystem    int8           `gorm:"not null;default:0" json:"isSystem"`
	Status      int8           `gorm:"not null;default:1;index" json:"status"`
	Remark      string         `gorm:"size:255" json:"remark"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SysConfig) TableName() string { return "sys_config" }
