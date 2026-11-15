package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type RoleController struct{ svc *service.RoleService }

func NewRoleController(database *gorm.DB) *RoleController {
	return &RoleController{svc: service.NewRoleService(database)}
}

// List 角色分页列表：GET /api/v1/roles?page=1&pageSize=20&keyword=
func (h *RoleController) List(c *gin.Context) {
	var q service.RoleQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, response.CodeParamInvalid, "查询参数格式错误")
		return
	}

	list, total, err := h.svc.List(&q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Page(c, list, total, q.Page, q.PageSize)
}

// Create 新增角色：POST /api/v1/roles
func (h *RoleController) Create(c *gin.Context) {
	var req service.RoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写角色名称与编码，且长度不得超过限制")
		return
	}

	id, err := h.svc.Create(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "新增角色", fmt.Sprintf("角色编码 %s，名称 %s", req.RoleCode, req.RoleName))
	response.OK(c, gin.H{"id": id})
}

// Update 修改角色：PUT /api/v1/roles/:id
func (h *RoleController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req service.RoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写角色名称与编码，且长度不得超过限制")
		return
	}
	if err := h.svc.Update(id, &req); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "修改角色", fmt.Sprintf("角色标识 %d，编码 %s", id, req.RoleCode))
	response.OK(c, nil)
}

// Delete 删除角色：DELETE /api/v1/roles/:id
func (h *RoleController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "删除角色", fmt.Sprintf("角色标识 %d", id))
	response.OK(c, nil)
}

// Permissions 查询某角色已授权的权限点标识：GET /api/v1/roles/:id/permissions
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

// Grant 角色授权：PUT /api/v1/roles/:id/permissions，整体替换该角色的权限集合
func (h *RoleController) Grant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		PermIDs []uint64 `json:"permIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "参数格式错误，permIds 不能为空")
		return
	}
	if err := h.svc.Grant(id, req.PermIDs); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "角色授权", fmt.Sprintf("角色标识 %d，授权 %d 个权限点", id, len(req.PermIDs)))
	response.OK(c, nil)
}
