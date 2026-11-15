package response

import "net/http"

// 业务错误码：与前端拦截器约定一致，0 表示成功
const (
	CodeSuccess      = 0
	CodeParamInvalid = 40001
	CodeUnauthorized = 40100
	CodeForbidden    = 40300
	CodeNotFound     = 40400
	CodeConflict     = 40900
	CodeServerError  = 50000
)

// HTTPStatus 业务码到 HTTP 状态码的映射
func HTTPStatus(code int) int {
	switch code {
	case CodeSuccess:
		return http.StatusOK
	case CodeParamInvalid:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
