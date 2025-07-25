// Package response 提供API响应相关的功能
package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/limitcool/starter/internal/errspec"
	"github.com/limitcool/starter/internal/pkg/errorx"
	"github.com/limitcool/starter/internal/pkg/logger"
)

// Result API标准响应结构
type Result[T any] struct {
	Code      int    `json:"code"`                 // 错误码
	Message   string `json:"message"`              // 提示信息
	Data      T      `json:"data"`                 // 数据
	RequestID string `json:"request_id,omitempty"` // 请求ID
	Time      int64  `json:"timestamp,omitempty"`  // 时间戳
	TraceID   string `json:"trace_id,omitempty"`   // 链路追踪ID
}

// PageResult 分页结果
type PageResult[T any] struct {
	Total    int64 `json:"total"`     // 总记录数
	Page     int   `json:"page"`      // 当前页码
	PageSize int   `json:"page_size"` // 每页大小
	List     T     `json:"list"`      // 数据列表
}

// NewPageResult 创建分页结果
func NewPageResult[T any](list T, total int64, page, pageSize int) *PageResult[T] {
	return &PageResult[T]{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     list,
	}
}

// Success 返回成功响应
func Success[T any](c *gin.Context, data T, msg ...string) {
	message := "success"
	if len(msg) > 0 {
		message = msg[0]
	}

	// 获取请求ID
	requestID := getRequestID(c)

	c.JSON(http.StatusOK, Result[T]{
		Code:      0, // 成功码为0
		Message:   message,
		Data:      data,
		RequestID: requestID,
		Time:      time.Now().Unix(),
	})
}

// SuccessNoData 返回无数据的成功响应
func SuccessNoData(c *gin.Context, msg ...string) {
	message := "success"
	if len(msg) > 0 {
		message = msg[0]
	}

	// 获取请求ID
	requestID := getRequestID(c)

	c.JSON(http.StatusOK, Result[struct{}]{
		Code:      0, // 成功码为0
		Message:   message,
		Data:      struct{}{},
		RequestID: requestID,
		Time:      time.Now().Unix(),
	})
}

// Error 返回错误响应
func Error(c *gin.Context, err error) {

	var (
		errorCode  = errspec.ErrUnknown.Code()
		httpStatus = http.StatusInternalServerError
		message    = err.Error()
	)

	// 获取请求上下文
	ctx := c.Request.Context()

	if e, ok := err.(interface{ Code() int }); ok {
		errorCode = e.Code()
	}

	if e, ok := err.(interface{ HttpStatus() int }); ok {
		httpStatus = e.HttpStatus()
	}

	// 获取请求ID
	requestID := getRequestID(c)

	// 获取链路追踪ID
	traceID := getTraceIDFromContext(c)

	// 记录错误到日志
	logger.ErrorContext(ctx, "API error occurred",
		"code", errorCode,
		"message", message,
		"trace_id", traceID,
		"request_id", requestID,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"client_ip", c.ClientIP(),
		"error_chain", errorx.FormatErrorChain(err),
	)

	// 统一响应结构
	c.JSON(httpStatus, Result[struct{}]{
		Code:      errorCode,
		Message:   message,
		Data:      struct{}{},
		RequestID: requestID,
		Time:      time.Now().Unix(),
		TraceID:   traceID,
	})
}

// getRequestID 获取请求ID，如果不存在则生成新的
func getRequestID(c *gin.Context) string {
	// 先从请求头部获取
	reqID := c.GetHeader("X-Request-ID")
	if reqID != "" {
		return reqID
	}

	// 如果上下文中已经有请求ID，则使用它
	if id, exists := c.Get("request_id"); exists {
		if strID, ok := id.(string); ok && strID != "" {
			return strID
		}
	}

	// 从请求上下文中获取
	ctx := c.Request.Context()
	if reqID, ok := ctx.Value("request_id").(string); ok && reqID != "" {
		return reqID
	}

	// 生成新的UUID作为请求ID
	newID := uuid.New().String()
	// 将请求ID存储到上下文中
	c.Set("request_id", newID)
	return newID
}

// getTraceIDFromContext 从上下文中获取链路追踪ID
func getTraceIDFromContext(c *gin.Context) string {
	// 先从上下文中获取
	if traceID, exists := c.Get("trace_id"); exists {
		if strID, ok := traceID.(string); ok && strID != "" {
			return strID
		}
	}

	// 从请求上下文中获取
	ctx := c.Request.Context()
	if traceID, ok := ctx.Value("trace_id").(string); ok && traceID != "" {
		return traceID
	}

	// 如果上下文中没有，尝试从请求头中获取
	traceID := c.GetHeader("X-Trace-ID")
	if traceID != "" {
		return traceID
	}

	return ""
}
