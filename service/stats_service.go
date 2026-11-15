package service

import (
	"time"

	"gorm.io/gorm"

	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

// StatsService 运营数据统计：输出 ECharts 可直接消费的 xAxis + series 结构
type StatsService struct{ db *gorm.DB }

func NewStatsService(database *gorm.DB) *StatsService { return &StatsService{db: database} }

// DaysParam 统一处理统计窗口：非法值回落到默认，超上限截断，避免一次查询扫全表
func DaysParam(raw int, def, max int) int {
	if raw <= 0 {
		return def
	}
	if raw > max {
		return max
	}
	return raw
}

// dateLabels 生成连续日期标签，统计结果按天补齐，缺失日期补 0
func dateLabels(days int) []string {
	labels := make([]string, 0, days)
	start := time.Now().AddDate(0, 0, -days+1)
	for i := 0; i < days; i++ {
		labels = append(labels, start.AddDate(0, 0, i).Format("2006-01-02"))
	}
	return labels
}

// monthLabels 生成连续月份标签（含当月），用于按月聚合
func monthLabels(months int) []string {
	labels := make([]string, 0, months)
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -months+1, 0)
	for i := 0; i < months; i++ {
		labels = append(labels, start.AddDate(0, i, 0).Format("2006-01"))
	}
	return labels
}

// UserGrowth 用户增长统计：新增用户数 + 累计用户数
type UserGrowth struct {
	Granularity string   `json:"granularity"`
	XAxis       []string `json:"xAxis"`
	NewUsers    []int64  `json:"newUsers"`
	TotalUsers  []int64  `json:"totalUsers"`
	SumNew      int64    `json:"sumNew"`
}

func (s *StatsService) UserGrowth(days int, granularity string) (*UserGrowth, error) {
	days = DaysParam(days, 30, 90)
	if granularity != "month" {
		granularity = "day"
	}

	var (
		labels   []string
		groupBy  string
		base     int64
		windowAt time.Time
	)

	if granularity == "day" {
		labels = dateLabels(days)
		groupBy = "DATE_FORMAT(created_at, '%Y-%m-%d')"
		windowAt = time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	} else {
		months := 12
		labels = monthLabels(months)
		groupBy = "DATE_FORMAT(created_at, '%Y-%m')"
		now := time.Now()
		windowAt = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).
			AddDate(0, -(months - 1), 0)
	}

	// 窗口之前的存量用户，作为累计曲线的起点
	if err := s.db.Model(&model.User{}).Where("created_at < ?", windowAt).
		Count(&base).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计存量用户失败", err)
	}

	var rows []TrendItem
	if err := s.db.Model(&model.User{}).
		Select(groupBy+" AS date, COUNT(*) AS count").
		Where("created_at >= ?", windowAt).
		Group(groupBy).Order("date ASC").Scan(&rows).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计新增用户失败", err)
	}

	byLabel := make(map[string]int64, len(rows))
	for _, r := range rows {
		byLabel[r.Date] = r.Count
	}

	res := &UserGrowth{Granularity: granularity, XAxis: labels,
		NewUsers: make([]int64, 0, len(labels)), TotalUsers: make([]int64, 0, len(labels))}
	running := base
	for _, l := range labels {
		v := byLabel[l]
		running += v
		res.SumNew += v
		res.NewUsers = append(res.NewUsers, v)
		res.TotalUsers = append(res.TotalUsers, running)
	}
	return res, nil
}

// UserActivity 用户活跃度：按天统计活跃用户、登录次数与操作总量
type UserActivity struct {
	XAxis        []string `json:"xAxis"`
	ActiveUsers  []int64  `json:"activeUsers"`
	LoginCount   []int64  `json:"loginCount"`
	OperationCnt []int64  `json:"operationCount"`
}

func (s *StatsService) UserActivity(days int) (*UserActivity, error) {
	days = DaysParam(days, 30, 90)
	labels := dateLabels(days)
	since := time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	groupBy := "DATE_FORMAT(created_at, '%Y-%m-%d')"

	var activeRows []TrendItem
	if err := s.db.Model(&model.OperationLog{}).
		Select(groupBy+" AS date, COUNT(DISTINCT user_id) AS count").
		Where("created_at >= ?", since).Group(groupBy).Order("date ASC").
		Scan(&activeRows).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计活跃用户失败", err)
	}

	var loginRows []TrendItem
	if err := s.db.Model(&model.OperationLog{}).
		Select(groupBy+" AS date, COUNT(*) AS count").
		Where("created_at >= ? AND module = ?", since, "登录鉴权").
		Group(groupBy).Order("date ASC").Scan(&loginRows).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计登录次数失败", err)
	}

	var opRows []TrendItem
	if err := s.db.Model(&model.OperationLog{}).
		Select(groupBy+" AS date, COUNT(*) AS count").
		Where("created_at >= ?", since).Group(groupBy).Order("date ASC").
		Scan(&opRows).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计操作总量失败", err)
	}

	pick := func(rows []TrendItem, label string) int64 {
		for _, r := range rows {
			if r.Date == label {
				return r.Count
			}
		}
		return 0
	}

	res := &UserActivity{XAxis: labels,
		ActiveUsers: make([]int64, 0, len(labels)), LoginCount: make([]int64, 0, len(labels)),
		OperationCnt: make([]int64, 0, len(labels))}
	for _, l := range labels {
		res.ActiveUsers = append(res.ActiveUsers, pick(activeRows, l))
		res.LoginCount = append(res.LoginCount, pick(loginRows, l))
		res.OperationCnt = append(res.OperationCnt, pick(opRows, l))
	}
	return res, nil
}

// LogFrequency 操作日志频次：总量、成功、失败按天聚合，并给出模块分布
type LogFrequency struct {
	XAxis   []string   `json:"xAxis"`
	Total   []int64    `json:"total"`
	Success []int64    `json:"success"`
	Failed  []int64    `json:"failed"`
	Module  []DistItem `json:"moduleDist"`
}

func (s *StatsService) LogFrequency(days int, module string) (*LogFrequency, error) {
	days = DaysParam(days, 7, 30)
	labels := dateLabels(days)
	since := time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	groupBy := "DATE_FORMAT(created_at, '%Y-%m-%d')"

	scope := func() *gorm.DB {
		tx := s.db.Model(&model.OperationLog{}).Where("created_at >= ?", since)
		if module != "" {
			tx = tx.Where("module = ?", module)
		}
		return tx
	}

	var totalRows, okRows, failRows []TrendItem
	if err := scope().Select(groupBy + " AS date, COUNT(*) AS count").
		Group(groupBy).Order("date ASC").Scan(&totalRows).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计日志总量失败", err)
	}
	if err := scope().Select(groupBy + " AS date, COUNT(*) AS count").
		Where("result = 1").Group(groupBy).Order("date ASC").Scan(&okRows).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计成功日志失败", err)
	}
	if err := scope().Select(groupBy + " AS date, COUNT(*) AS count").
		Where("result = 0").Group(groupBy).Order("date ASC").Scan(&failRows).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计失败日志失败", err)
	}

	var dist []DistItem
	if err := scope().Select("module AS name, COUNT(*) AS count").
		Group("module").Order("count DESC").Limit(10).Scan(&dist).Error; err != nil {
		return nil, utils.WrapAppError(response.CodeServerError, "统计模块分布失败", err)
	}

	pick := func(rows []TrendItem, label string) int64 {
		for _, r := range rows {
			if r.Date == label {
				return r.Count
			}
		}
		return 0
	}

	res := &LogFrequency{XAxis: labels, Total: make([]int64, 0, len(labels)),
		Success: make([]int64, 0, len(labels)), Failed: make([]int64, 0, len(labels)), Module: dist}
	for _, l := range labels {
		res.Total = append(res.Total, pick(totalRows, l))
		res.Success = append(res.Success, pick(okRows, l))
		res.Failed = append(res.Failed, pick(failRows, l))
	}

	// 兜底：模块分布为空时返回空数组而不是 null，前端不必判空
	if res.Module == nil {
		res.Module = []DistItem{}
	}
	return res, nil
}
