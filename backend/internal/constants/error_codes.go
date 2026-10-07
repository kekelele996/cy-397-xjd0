package constants

// 统一响应业务错误码。0 表示成功，其余为业务/系统错误。
const (
	CodeSuccess           = 0
	CodeInternalError     = 50000
	CodeBadRequest        = 40000
	CodeValidationFailed  = 40001
	CodeUnauthorized      = 40100
	CodeTokenInvalid      = 40101
	CodeForbidden         = 40300
	CodeNotFound          = 40400
	CodeConflict          = 40900
	CodeInvalidTransition = 42200
)
