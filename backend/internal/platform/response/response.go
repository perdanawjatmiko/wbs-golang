package response

import "github.com/gin-gonic/gin"

type Meta struct {
	RequestID string `json:"request_id"`
}
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Envelope struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
	Meta    Meta      `json:"meta"`
}

func Success(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{Success: true, Data: data, Meta: Meta{c.GetString("request_id")}})
}
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, Envelope{Error: &APIError{code, message}, Meta: Meta{c.GetString("request_id")}})
}
