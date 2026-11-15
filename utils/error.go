package utils

import "fmt"

// AppError 可预期的业务错误，携带业务码与可直接展示的提示
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func NewAppError(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func WrapAppError(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
