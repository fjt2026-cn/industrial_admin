package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type DashboardController struct{ svc *service.DashboardService }

func NewDashboardController(database *gorm.DB) *DashboardController {
	return &DashboardController{svc: service.NewDashboardService(database)}
}

// Overview 运营看板：GET /api/v1/dashboard/overview?days=7
func (h *DashboardController) Overview(c *gin.Context) {
	days := 7
	if raw := c.Query("days"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			days = v
		}
	}

	ov, err := h.svc.Overview(days)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, ov)
}
