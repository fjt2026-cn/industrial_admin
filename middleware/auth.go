package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/config"
	"industrial_admin/db"
	"industrial_admin/model"
	"industrial_admin/response"
	"industrial_admin/utils"
)

// 上下文键，供控制器与权限中间件读取
const (
	CtxUserID   = "ctx_user_id"
	CtxUsername = "ctx_username"
)

// Auth 鉴权：解析 Bearer 令牌，校验签名、过期与用户状态
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			response.Abort(c, response.CodeUnauthorized, "未登录或令牌缺失")
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := utils.ParseToken(cfg, tokenStr)
		if err != nil {
			// 区分“过期”与“无效”：过期提示用户重新登录更友好
			msg := "令牌无效，请重新登录"
			if errors.Is(err, utils.ErrTokenExpired) {
				msg = "登录已失效，请重新登录"
			}
			response.Abort(c, response.CodeUnauthorized, msg)
			return
		}

		// 令牌有效不代表账号仍可用：停用或删除后即时失效
		var u model.User
		if err := db.DB.Where("id = ?", claims.UserID).First(&u).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Abort(c, response.CodeUnauthorized, "账号不存在")
				return
			}
			response.Abort(c, response.CodeServerError, "服务器内部错误")
			return
		}
		if u.Status != 1 {
			response.Abort(c, response.CodeUnauthorized, "账号已停用，请联系管理员")
			return
		}

		c.Set(CtxUserID, u.ID)
		c.Set(CtxUsername, u.Username)
		c.Next()
	}
}

// CurrentUserID 从上下文取当前用户标识
func CurrentUserID(c *gin.Context) uint64 {
	if v, ok := c.Get(CtxUserID); ok {
		if id, ok := v.(uint64); ok {
			return id
		}
	}
	return 0
}

// CurrentUsername 从上下文取当前账号
func CurrentUsername(c *gin.Context) string {
	if v, ok := c.Get(CtxUsername); ok {
		if name, ok := v.(string); ok {
			return name
		}
	}
	return ""
}
