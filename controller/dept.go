package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
)

type DeptController struct{ svc *service.DeptService }

func NewDeptController(database *gorm.DB) *DeptController {
	return &DeptController{svc: service.NewDeptService(database)}
}

// Tree 部门树
func (h *DeptController) Tree(c *gin.Context) {
	tree, err := h.svc.Tree()
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tree)
}

// Create 新增部门
func (h *DeptController) Create(c *gin.Context) {
	var req service.CreateDeptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写部门名称与编码")
		return
	}
	id, err := h.svc.Create(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}

// Remove 删除部门（存在子部门或成员时拒绝）
func (h *DeptController) Remove(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Remove(id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}
