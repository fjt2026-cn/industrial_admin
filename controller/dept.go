package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type DeptController struct{ svc *service.DeptService }

func NewDeptController(database *gorm.DB) *DeptController {
	return &DeptController{svc: service.NewDeptService(database)}
}

// Tree 部门树：GET /api/v1/depts/tree
func (h *DeptController) Tree(c *gin.Context) {
	tree, err := h.svc.Tree()
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tree)
}

// List 部门平铺分页列表：GET /api/v1/depts
func (h *DeptController) List(c *gin.Context) {
	var q service.DeptQuery
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

// Detail 部门详情：GET /api/v1/depts/:id
func (h *DeptController) Detail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	d, parentName, err := h.svc.Detail(id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"dept": d, "parentName": parentName})
}

// Create 新增部门：POST /api/v1/depts
func (h *DeptController) Create(c *gin.Context) {
	var req service.CreateDeptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写部门名称与编码，且长度不得超过限制")
		return
	}
	id, err := h.svc.Create(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "新增部门", fmt.Sprintf("部门编码 %s，名称 %s", req.DeptCode, req.DeptName))
	response.OK(c, gin.H{"id": id})
}

// Update 修改部门：PUT /api/v1/depts/:id
func (h *DeptController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req service.CreateDeptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写部门名称与编码，且长度不得超过限制")
		return
	}
	if err := h.svc.Update(id, &req); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "修改部门", fmt.Sprintf("部门标识 %d，编码 %s", id, req.DeptCode))
	response.OK(c, nil)
}

// Remove 删除部门：DELETE /api/v1/depts/:id，存在子部门或成员时拒绝
func (h *DeptController) Remove(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Remove(id); err != nil {
		response.Error(c, err)
		return
	}
	logKeyOp(c, "删除部门", fmt.Sprintf("部门标识 %d", id))
	response.OK(c, nil)
}
