package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type ConfigController struct{ svc *service.ConfigService }

func NewConfigController(database *gorm.DB) *ConfigController {
	return &ConfigController{svc: service.NewConfigService(database)}
}

// List 按分类查询参数：GET /api/v1/configs?category=
func (h *ConfigController) List(c *gin.Context) {
	rows, err := h.svc.List(c.Query("category"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}

// Detail 参数详情：GET /api/v1/configs/:id
func (h *ConfigController) Detail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	row, err := h.svc.Detail(id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, row)
}

// Update 修改参数值，成功后刷新内存缓存：PUT /api/v1/configs/:id
func (h *ConfigController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		ConfigValue string `json:"configValue"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "参数值格式错误")
		return
	}
	if err := h.svc.Update(id, req.ConfigValue); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "修改系统参数", fmt.Sprintf("参数标识 %d，新值 %s", id, req.ConfigValue))
	response.OK(c, nil)
}

// BatchUpdate 批量修改参数：PUT /api/v1/configs，事务内全部成功才对缓存生效
func (h *ConfigController) BatchUpdate(c *gin.Context) {
	var req struct {
		Items []service.ConfigItem `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "参数格式错误，items 不能为空")
		return
	}
	updated, err := h.svc.BatchUpdate(req.Items)
	if err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "批量修改系统参数", fmt.Sprintf("共修改 %d 项", updated))
	response.OK(c, gin.H{"updated": updated})
}
