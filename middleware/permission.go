package middleware

import (
	"github.com/gin-gonic/gin"

	"industrial_admin/db"
	"industrial_admin/model"
	"industrial_admin/response"
)

// RequirePerm RBAC 接口级鉴权：权限编码命中即放行，否则 403
func RequirePerm(permCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := CurrentUserID(c)
		if userID == 0 {
			response.Abort(c, response.CodeUnauthorized, "未登录")
			return
		}

		var count int64
		err := db.DB.Table("permission p").
			Joins("JOIN role_permission rp ON rp.perm_id = p.id").
			Joins("JOIN user_role ur ON ur.role_id = rp.role_id").
			Where("ur.user_id = ? AND p.perm_code = ? AND p.status = 1 AND p.deleted_at IS NULL", userID, permCode).
			Count(&count).Error
		if err != nil {
			response.Abort(c, response.CodeServerError, "权限校验失败")
			return
		}
		if count == 0 {
			// 越权访问单独记一条日志，便于发现异常探测
			db.DB.Create(&model.OperationLog{
				UserID:   &userID,
				Username: CurrentUsername(c),
				Module:   "auth",
				Action:   "forbidden",
				Method:   c.Request.Method,
				URL:      c.Request.URL.Path,
				IP:       c.ClientIP(),
				Result:   0,
				ErrorMsg: "权限不足：" + permCode,
			})
			response.Abort(c, response.CodeForbidden, "权限不足")
			return
		}
		c.Next()
	}
}
