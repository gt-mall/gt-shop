package response

import "github.com/gin-gonic/gin"

// Response is the only response envelope used by controllers.
type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Total   int    `json:"total"`
	Data    T      `json:"data"`
}

func Success(c *gin.Context) {
	JSON(c, Response[any]{Code: 1, Message: "ok", Data: nil})
}

func Error(c *gin.Context, message string) {
	JSON(c, Response[any]{Code: 0, Message: message, Data: nil})
}

func Unauthorized(c *gin.Context, message string) {
	JSON(c, Response[any]{Code: -1, Message: message, Data: nil})
}

func Data[T any](c *gin.Context, data T, total int) {
	JSON(c, Response[T]{Code: 1, Message: "ok", Total: total, Data: data})
}

func JSON[T any](c *gin.Context, result Response[T]) {
	c.JSON(200, result)
}
