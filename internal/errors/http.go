package errors


// ForbiddenError 403 禁止访问
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return e.Message
}


// BadRequestError 400 请求错误
type BadRequestError struct {
	Message string
}

func (e *BadRequestError) Error() string {
	return e.Message
}
