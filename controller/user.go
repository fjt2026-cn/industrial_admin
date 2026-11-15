package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/response"
	"industrial_admin/service"
	"industrial_admin/utils"
)

type UserController struct{ svc *service.UserService }

func NewUserController(database *gorm.DB) *UserController {
	return &UserController{svc: service.NewUserService(database)}
}

// List 用户分页列表
func (h *UserController) List(c *gin.Context) {
	var q service.UserQuery
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

// Create 新增用户（含角色分配，事务提交）
func (h *UserController) Create(c *gin.Context) {
	var req service.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "请填写账号、口令与姓名")
		return
	}

	// 口令策略在入口统一校验，避免短口令走到哈希环节才报 500
	if err := utils.ValidatePassword(req.Password); err != nil {
		response.Fail(c, response.CodeParamInvalid, err.Error())
		return
	}

	id, err := h.svc.Create(&req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}

// Update 编辑用户
func (h *UserController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req service.UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "参数格式错误")
		return
	}
	if err := h.svc.Update(id, &req); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete 删除用户（软删除并释放账号唯一值）
func (h *UserController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ResetPassword 管理员重置口令
func (h *UserController) ResetPassword(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeParamInvalid, "新口令不能为空")
		return
	}
	if err := utils.ValidatePassword(req.Password); err != nil {
		response.Fail(c, response.CodeParamInvalid, err.Error())
		return
	}
	if err := h.svc.ResetPassword(id, req.Password); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// parseID 解析路径参数 id，失败时直接返回统一错误
func parseID(c *gin.Context) (uint64, bool) {
	raw := c.Param("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, response.CodeParamInvalid, "ID 不合法")
		return 0, false
	}
	return id, true
}
