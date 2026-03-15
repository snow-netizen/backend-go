package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 标准响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	Response
	Pagination Pagination `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	Total     int64 `json:"total"`
	TotalPage int64 `json:"total_page"`
}

// NewPageResponse 创建分页响应
func NewPageResponse(code int, message string, data interface{}, page, pageSize int, total int64) PageResponse {
	totalPage := total / int64(pageSize)
	if total%int64(pageSize) > 0 {
		totalPage++
	}
	
	return PageResponse{
		Response: Response{
			Code:    code,
			Message: message,
			Data:    data,
		},
		Pagination: Pagination{
			Page:      page,
			PageSize:  pageSize,
			Total:     total,
			TotalPage: totalPage,
		},
	}
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: message,
		Data:    data,
	})
}

// SuccessPage 分页成功响应
func SuccessPage(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	c.JSON(http.StatusOK, NewPageResponse(200, "success", data, page, pageSize, total))
}

// Created 创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    201,
		Message: "created",
		Data:    data,
	})
}

// BadRequest 错误请求响应
func BadRequest(c *gin.Context, err string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    400,
		Message: "bad request",
		Error:   err,
	})
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, err string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    401,
		Message: "unauthorized",
		Error:   err,
	})
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, err string) {
	c.JSON(http.StatusForbidden, Response{
		Code:    403,
		Message: "forbidden",
		Error:   err,
	})
}

// NotFound 未找到响应
func NotFound(c *gin.Context, err string) {
	c.JSON(http.StatusNotFound, Response{
		Code:    404,
		Message: "not found",
		Error:   err,
	})
}

// InternalServerError 服务器内部错误响应
func InternalServerError(c *gin.Context, err string) {
	c.JSON(http.StatusInternalServerError, Response{
		Code:    500,
		Message: "internal server error",
		Error:   err,
	})
}

// CustomError 自定义错误响应
func CustomError(c *gin.Context, code int, message, err string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Error:   err,
	})
}