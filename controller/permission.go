package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type PermissionController struct{ svc *service.PermissionService }

func NewPermissionController(database *gorm.DB) *PermissionController {
	return &PermissionController{svc: service.NewPermissionService(database)}
}

// List 权限点分页列表：GET /api/v1/permissions?page=1&pageSize=20&keyword=&permType=
func (h *PermissionController) List(c *gin.Context) {
	var q service.PermissionQuery
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

// Create 新增权限点：POST /api/v1/permissions
func (h *PermissionController) Create(c *gin.Context) {
	var req service.PermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写权限名称、编码与类型（1 菜单 / 2 按钮 / 3 接口）")
		return
	}

	id, err := h.svc.Create(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "新增权限点", fmt.Sprintf("权限编码 %s，名称 %s", req.PermCode, req.PermName))
	response.OK(c, gin.H{"id": id})
}

// Update 修改权限点：PUT /api/v1/permissions/:id
func (h *PermissionController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req service.PermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写权限名称、编码与类型")
		return
	}
	if err := h.svc.Update(id, &req); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "修改权限点", fmt.Sprintf("权限标识 %d，编码 %s", id, req.PermCode))
	response.OK(c, nil)
}

// Delete 删除权限点：DELETE /api/v1/permissions/:id
func (h *PermissionController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "删除权限点", fmt.Sprintf("权限标识 %d", id))
	response.OK(c, nil)
}
