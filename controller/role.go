package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type RoleController struct{ svc *service.RoleService }

func NewRoleController(database *gorm.DB) *RoleController {
	return &RoleController{svc: service.NewRoleService(database)}
}

// List 角色列表
func (h *RoleController) List(c *gin.Context) {
	roles, err := h.svc.List()
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, roles)
}

// Create 新增角色
func (h *RoleController) Create(c *gin.Context) {
	var req struct {
		RoleName  string `json:"roleName" binding:"required"`
		RoleCode  string `json:"roleCode" binding:"required"`
		DataScope int8   `json:"dataScope"`
		Remark    string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写角色名称与编码")
		return
	}
	if req.DataScope == 0 {
		req.DataScope = 1
	}
	id, err := h.svc.Create(req.RoleName, req.RoleCode, req.DataScope, req.Remark)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}

// Permissions 查询某角色已授权的权限点标识
func (h *RoleController) Permissions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ids, err := h.svc.PermissionIDs(id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, ids)
}

// Grant 角色授权：整体替换该角色的权限集合
func (h *RoleController) Grant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		PermIDs []uint64 `json:"permIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "参数格式错误")
		return
	}
	if err := h.svc.Grant(id, req.PermIDs); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}
