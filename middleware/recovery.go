package middleware

import (
	"log"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"industrial_admin/response"
)

// Recovery 全局异常处理：捕获 panic、打印堆栈、返回统一结构，进程不退出
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Printf("panic recovered: %v\n%s", err, debug.Stack())
		response.Abort(c, response.CodeServerError, "服务器内部错误，请联系管理员")
	})
}
