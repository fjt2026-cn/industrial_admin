package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/config"
	"industrial_admin/middleware"
	"industrial_admin/response"
	"industrial_admin/service"
)

type AuthController struct{ svc *service.AuthService }

func NewAuthController(cfg *config.Config, database *gorm.DB) *AuthController {
	return &AuthController{svc: service.NewAuthService(database, cfg)}
}

// Login 登录接口：POST /api/v1/auth/login
func (h *AuthController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,max=50"`
		Password string `json:"password" binding:"required,max=72"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 只提示“不能为空或长度超限”，不回显具体字段，避免为探测者提供信息
		response.Fail(c, response.CodeParamInvalid, "账号与口令不能为空，且长度不得超过限制")
		return
	}

	res, err := h.svc.Login(strings.TrimSpace(req.Username), req.Password, c.ClientIP())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// Profile 当前用户信息与菜单：GET /api/v1/auth/profile
func (h *AuthController) Profile(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	info, perms, menus, err := h.svc.Profile(uid)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"user": info, "perms": perms, "menus": menus})
}
