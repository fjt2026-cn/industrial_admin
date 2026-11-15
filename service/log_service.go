package service

import (
	"time"

	"gorm.io/gorm"

	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

type LogService struct{ db *gorm.DB }

func NewLogService(database *gorm.DB) *LogService { return &LogService{db: database} }

// LogQuery 日志检索条件：操作人、模块、结果、时间范围与关键字
type LogQuery struct {
	utils.PageQuery
	Username  string `form:"username"`
	Module    string `form:"module"`
	Result    *int8  `form:"result"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
	Keyword   string `form:"keyword"`
}

func (s *LogService) List(q *LogQuery) ([]model.OperationLog, int64, error) {
	q.Normalize()

	tx := s.db.Model(&model.OperationLog{})
	if q.Username != "" {
		tx = tx.Where("username = ?", q.Username)
	}
	if q.Module != "" {
		tx = tx.Where("module = ?", q.Module)
	}
	if q.Result != nil {
		tx = tx.Where("result = ?", *q.Result)
	}
	if q.StartTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", q.StartTime); err == nil {
			tx = tx.Where("created_at >= ?", t)
		}
	}
	if q.EndTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", q.EndTime); err == nil {
			tx = tx.Where("created_at <= ?", t)
		}
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("url LIKE ? OR error_msg LIKE ?", kw, kw)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询日志总数失败", err)
	}

	var rows []model.OperationLog
	if err := tx.Order("id DESC").Offset(q.Offset()).Limit(q.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, utils.WrapAppError(response.CodeServerError, "查询日志失败", err)
	}
	return rows, total, nil
}
