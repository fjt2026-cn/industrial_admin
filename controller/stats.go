package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type StatsController struct{ svc *service.StatsService }

func NewStatsController(database *gorm.DB) *StatsController {
	return &StatsController{svc: service.NewStatsService(database)}
}

func queryInt(c *gin.Context, key string) int {
	raw := c.Query(key)
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return v
}

// UserGrowth 用户增长统计：GET /api/v1/stats/users/growth?days=30&granularity=day|month
func (h *StatsController) UserGrowth(c *gin.Context) {
	res, err := h.svc.UserGrowth(queryInt(c, "days"), c.DefaultQuery("granularity", "day"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// UserActivity 用户活跃度：GET /api/v1/stats/users/activity?days=30
func (h *StatsController) UserActivity(c *gin.Context) {
	res, err := h.svc.UserActivity(queryInt(c, "days"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// LogFrequency 操作日志频次：GET /api/v1/stats/logs/frequency?days=7&module=
func (h *StatsController) LogFrequency(c *gin.Context) {
	res, err := h.svc.LogFrequency(queryInt(c, "days"), c.Query("module"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}
