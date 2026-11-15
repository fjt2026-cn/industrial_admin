package service

import (
	"time"

	"gorm.io/gorm"

	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

type DashboardService struct{ db *gorm.DB }

func NewDashboardService(database *gorm.DB) *DashboardService {
	return &DashboardService{db: database}
}

type TrendItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type DistItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type Overview struct {
	UserTotal   int64       `json:"userTotal"`
	TodayActive int64       `json:"todayActive"`
	TodayOps    int64       `json:"todayOps"`
	FailOps     int64       `json:"failOps"`
	UserTrend   []TrendItem `json:"userTrend"`
	ModuleDist  []DistItem  `json:"moduleDist"`
}

// Overview 看板聚合：一次请求返回概览、趋势与分布，减少往返
func (s *DashboardService) Overview(days int) (*Overview, error) {
	if days != 7 && days != 30 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	todayStart := time.Now().Truncate(24 * time.Hour)

	ov := &Overview{UserTrend: make([]TrendItem, 0), ModuleDist: make([]DistItem, 0)}

	if err := s.db.Model(&model.User{}).Count(&ov.UserTotal).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计用户总数失败", err)
	}

	if err := s.db.Model(&model.OperationLog{}).
		Where("created_at >= ?", todayStart).
		Distinct("user_id").Count(&ov.TodayActive).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计活跃用户失败", err)
	}

	if err := s.db.Model(&model.OperationLog{}).
		Where("created_at >= ?", todayStart).Count(&ov.TodayOps).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计操作数失败", err)
	}

	if err := s.db.Model(&model.OperationLog{}).
		Where("created_at >= ? AND result = 0", todayStart).Count(&ov.FailOps).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计失败操作失败", err)
	}

	// 用户增长趋势：按创建日期分组，缺失日期由前端补零
	if err := s.db.Model(&model.User{}).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d') AS date, COUNT(*) AS count").
		Where("created_at >= ?", since).
		Group("DATE_FORMAT(created_at, '%Y-%m-%d')").
		Order("date ASC").Scan(&ov.UserTrend).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计用户趋势失败", err)
	}

	// 功能使用分布：按模块统计操作条数
	if err := s.db.Model(&model.OperationLog{}).
		Select("module AS name, COUNT(*) AS count").
		Where("created_at >= ?", since).
		Group("module").Order("count DESC").Scan(&ov.ModuleDist).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计功能分布失败", err)
	}

	return ov, nil
}
