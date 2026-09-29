package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 200,
		Msg:  "success",
		Data: data,
	})
}

func SuccessMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{
		Code: 200,
		Msg:  msg,
	})
}


// Error 错误统一出口：HTTP 状态码与 body code 严格一致（双轨）
func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, Response{
		Code: code,
		Msg:  msg,
	})
}

// ErrorData 错误响应携带 data（如代理上游失败时回传子节点原始响应体）
func ErrorData(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(code, Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

func BadRequest(c *gin.Context, msg string) {
	Error(c, 400, msg)
}

func Unauthorized(c *gin.Context, msg string) {
	Error(c, 401, msg)
}

func Forbidden(c *gin.Context, msg string) {
	Error(c, 403, msg)
}

func NotFound(c *gin.Context, msg string) {
	Error(c, 404, msg)
}


func TooManyRequests(c *gin.Context, msg string) {
	Error(c, 429, msg)
}

func ServerError(c *gin.Context, msg string) {
	Error(c, 500, msg)
}
