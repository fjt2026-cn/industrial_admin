package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type LogController struct{ svc *service.LogService }

func NewLogController(database *gorm.DB) *LogController {
	return &LogController{svc: service.NewLogService(database)}
}

// List 操作日志分页检索
func (h *LogController) List(c *gin.Context) {
	var q service.LogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.CodeParamInvalid, "查询参数格式错误")
		return
	}

	rows, total, err := h.svc.List(&q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Page(c, rows, total, q.Page, q.PageSize)
}
