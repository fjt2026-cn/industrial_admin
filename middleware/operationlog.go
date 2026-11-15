package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"industrial_admin/model"
)

// OperationLog 关键操作留痕：在响应返回后异步落库，失败不影响业务
func OperationLog(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// 只记录写操作与登录，查询不落库，避免日志表被读操作淹没
		if c.Request.Method == "GET" {
			return
		}
		userID := CurrentUserID(c)
		entry := model.OperationLog{
			Username:  CurrentUsername(c),
			Module:    c.FullPath(),
			Action:    c.Request.Method,
			Method:    c.Request.Method,
			URL:       c.Request.URL.Path,
			IP:        c.ClientIP(),
			CostMs:    int(time.Since(start).Milliseconds()),
			UserAgent: c.Request.UserAgent(),
			Result:    1,
		}
		if userID != 0 {
			uid := userID
			entry.UserID = &uid
		}
		if c.Writer.Status() >= 400 {
			entry.Result = 0
			entry.ErrorMsg = c.Writer.Header().Get("X-Error-Msg")
		}

		go func(e model.OperationLog) {
			if err := database.Create(&e).Error; err != nil {
				// 日志失败只打点，不回滚业务
				_ = err
			}
		}(entry)
	}
}
