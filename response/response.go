package response

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"industrial_admin/utils"
)

// Body 是全项目唯一的响应外壳，结构固定为 {code, message, data}。
// 额外附带 timestamp，便于前端排查“旧响应被缓存”的问题。
// 约定：任何接口都不得自行拼装 JSON，一律通过本文件的函数输出。
type Body struct {
	Code      int    `json:"code"`      // 业务码，0 表示成功
	Message   string `json:"message"`   // 提示文案，可直接展示
	Data      any    `json:"data"`      // 业务数据，失败时为 null
	Timestamp int64  `json:"timestamp"` // 服务器时间戳（毫秒）
}

// PageData 是列表接口的标准数据体。
// 前端表格组件只需读取 list 渲染行、total 计算分页总数。
type PageData struct {
	List     any   `json:"list"`     // 当前页数据
	Total    int64 `json:"total"`    // 记录总数
	Page     int   `json:"page"`     // 当前页码，从 1 开始
	PageSize int   `json:"pageSize"` // 每页条数
}

// OK 输出成功响应，业务码固定为 0。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code:      CodeSuccess,
		Message:   "ok",
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	})
}

// Success 与 OK 等价，保留两个命名便于在代码中表达“查询成功/操作成功”的语义。
func Success(c *gin.Context, data any) { OK(c, data) }

// Page 输出分页成功响应，把 list、total、page、pageSize 收进 data。
func Page(c *gin.Context, list any, total int64, page, pageSize int) {
	OK(c, PageData{List: list, Total: total, Page: page, PageSize: pageSize})
}

// Abort 输出失败响应并终止后续处理，专供中间件使用。
// 与 Fail 的差别：AbortWithStatusJSON 会阻止后续 handler 继续执行，
// 否则鉴权失败后仍会进入业务逻辑，形成越权风险。
func Abort(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(HTTPStatus(code), Body{
		Code:      code,
		Message:   message,
		Data:      nil,
		Timestamp: time.Now().UnixMilli(),
	})
}

// Fail 输出失败响应，供控制器使用。
// HTTP 状态码由业务码映射得到，保证前端既能按状态码拦截，也能按业务码细分提示。
func Fail(c *gin.Context, code int, message string) {
	c.JSON(HTTPStatus(code), Body{
		Code:      code,
		Message:   message,
		Data:      nil,
		Timestamp: time.Now().UnixMilli(),
	})
}

// Error 是控制器的统一错误出口。
// service 层返回 *utils.AppError 时按其错误码输出；其余（数据库报错、空指针等）
// 统一收敛成 50000，绝不把底层错误细节暴露给调用方。
func Error(c *gin.Context, err error) {
	var ae *utils.AppError
	if errors.As(err, &ae) {
		Fail(c, ae.Code, ae.Message)
		return
	}
	Fail(c, CodeServerError, "服务器内部错误")
}
