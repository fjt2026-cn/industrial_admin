package controller

import (
	"github.com/gin-gonic/gin"

	"industrial_admin/db"
	"industrial_admin/middleware"
	"industrial_admin/model"
)

// logKeyOp 记录关键业务操作的细节（新增用户、角色授权、角色绑定等）
// 与操作留痕中间件互补：中间件记“请求级”日志，这里记“业务语义级”日志
func logKeyOp(c *gin.Context, action, detail string) {
	entry := model.OperationLog{
		Username: middleware.CurrentUsername(c),
		Module:   "RBAC",
		Action:   action,
		Method:   c.Request.Method,
		URL:      c.Request.URL.Path,
		IP:       c.ClientIP(),
		Result:   1,
		Param:    detail,
	}
	if uid := middleware.CurrentUserID(c); uid != 0 {
		entry.UserID = &uid
	}
	// 日志写入失败不影响业务结果
	_ = db.DB.Create(&entry).Error
}
