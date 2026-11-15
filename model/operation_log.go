package model

import "time"

// OperationLog 操作日志：只追加，不更新、不软删除
type OperationLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *uint64   `gorm:"index" json:"userId"`
	Username  string    `gorm:"size:50" json:"username"`
	Module    string    `gorm:"size:50;not null" json:"module"`
	Action    string    `gorm:"size:50;not null" json:"action"`
	Method    string    `gorm:"size:10" json:"method"`
	URL       string    `gorm:"column:url;size:255" json:"url"`
	IP        string    `gorm:"column:ip;size:45" json:"ip"`
	Param     string    `gorm:"type:text" json:"param"`
	Result    int8      `gorm:"not null;index" json:"result"`
	ErrorMsg  string    `gorm:"size:500" json:"errorMsg"`
	CostMs    int       `gorm:"not null;default:0" json:"costMs"`
	UserAgent string    `gorm:"size:255" json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
}

func (OperationLog) TableName() string { return "operation_log" }
