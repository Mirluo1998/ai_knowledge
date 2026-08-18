package service

// ValidationError 表示由客户端输入引起的业务校验错误。
// handler 层通过 errors.As 识别该类型并返回 4xx 状态码，
// 其余错误一律视为内部错误返回 5xx。
type ValidationError struct {
	Message string
}

// Error 实现 error 接口。
func (e *ValidationError) Error() string {
	return e.Message
}
