package dto

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/contractapi/contractapi/internal/constants"
)

// AppError 业务错误，包含 HTTP 状态码、业务码与用户可读信息。
type AppError struct {
	Status  int
	Code    int
	Message string
	// Data 可选的结构化错误明细（如关单对账差异列表）。
	Data any
	Err  error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// WithData 附带结构化错误明细。
func (e *AppError) WithData(data any) *AppError {
	e.Data = data
	return e
}

// NewAppError 构造业务错误。
func NewAppError(status, code int, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// WrapAppError 包装底层错误并保留错误链。
func WrapAppError(status, code int, message string, err error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: err}
}

// ValidationError 构造参数校验错误。
func ValidationError(message string) *AppError {
	return NewAppError(http.StatusBadRequest, constants.CodeValidationFailed, message)
}

// UnauthorizedError 构造认证失败错误。
func UnauthorizedError(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, constants.CodeUnauthorized, message)
}

// NotFoundError 构造资源不存在错误。
func NotFoundError(message string) *AppError {
	return NewAppError(http.StatusNotFound, constants.CodeNotFound, message)
}

// ConflictError 构造冲突错误。
func ConflictError(message string) *AppError {
	return NewAppError(http.StatusConflict, constants.CodeConflict, message)
}

// InvalidTransitionError 构造非法状态流转错误。
func InvalidTransitionError(message string) *AppError {
	return NewAppError(http.StatusUnprocessableEntity, constants.CodeInvalidTransition, message)
}

// ReconcileMismatchError 构造关单对账不一致错误，mismatches 为差异明细。
func ReconcileMismatchError(message string, data any) *AppError {
	return (&AppError{
		Status:  http.StatusConflict,
		Code:    constants.CodeReconcileMismatch,
		Message: message,
		Data:    data,
	})
}

// InternalError 构造内部错误。
func InternalError(err error) *AppError {
	return WrapAppError(http.StatusInternalServerError, constants.CodeInternalError, "internal server error", err)
}

// IsAppError 判断错误链中是否包含 AppError。
func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
