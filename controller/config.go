package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type ConfigController struct{ svc *service.ConfigService }

func NewConfigController(database *gorm.DB) *ConfigController {
	return &ConfigController{svc: service.NewConfigService(database)}
}

// List 按分类查询参数
func (h *ConfigController) List(c *gin.Context) {
	rows, err := h.svc.List(c.Query("category"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}

// Update 修改参数值，成功后刷新内存缓存
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
	response.OK(c, nil)
}
