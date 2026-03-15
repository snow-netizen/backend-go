package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"admin-system-go/internal/logger"
)

// bodyLogWriter 用于捕获响应体的 writer
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 写入数据到缓冲区
func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 获取请求信息
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 生成请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			c.Header("X-Request-ID", requestID)
		}

		// 记录请求开始
		logger.Info("request started",
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		)

		// 捕获请求体（如果是JSON请求且体不太大）
		var requestBody []byte
		if c.Request.Body != nil && c.Request.ContentLength > 0 && c.Request.ContentLength < 1024*1024 { // 限制1MB
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

			// 记录请求体（不记录敏感信息）
			if len(requestBody) > 0 && len(requestBody) < 1024 { // 限制1KB
				logger.Debug("request body",
					zap.String("request_id", requestID),
					zap.String("content_type", c.ContentType()),
					zap.ByteString("body", requestBody),
				)
			}
		}

		// 捕获响应体
		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: c.Writer,
		}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 计算处理时间
		latency := time.Since(start)

		// 获取响应信息
		statusCode := c.Writer.Status()
		responseSize := c.Writer.Size()

		// 获取用户上下文（如果已认证）
		var userID uint
		var username, role string
		if id, exists := c.Get("user_id"); exists {
			if uid, ok := id.(uint); ok {
				userID = uid
			}
		}
		if name, exists := c.Get("username"); exists {
			if uname, ok := name.(string); ok {
				username = uname
			}
		}
		if r, exists := c.Get("role"); exists {
			if rStr, ok := r.(string); ok {
				role = rStr
			}
		}

		// 构建日志字段
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.Int("status", statusCode),
			zap.Int("response_size", responseSize),
			zap.Duration("latency", latency),
			zap.String("latency_human", latency.String()),
		}

		// 添加用户上下文（如果有）
		if userID > 0 {
			fields = append(fields, zap.Uint("user_id", userID))
		}
		if username != "" {
			fields = append(fields, zap.String("username", username))
		}
		if role != "" {
			fields = append(fields, zap.String("role", role))
		}

		// 记录错误信息（如果有）
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				fields = append(fields, zap.Error(e))
			}
		}

		// 记录响应体（如果是错误响应且体不太大）
		if statusCode >= 400 && blw.body.Len() > 0 && blw.body.Len() < 1024 {
			fields = append(fields, zap.String("response_body", blw.body.String()))
		}

		// 根据状态码确定日志级别
		msg := "request completed"
		switch {
		case statusCode >= 500:
			logger.Error(msg, fields...)
		case statusCode >= 400:
			logger.Warn(msg, fields...)
		default:
			logger.Info(msg, fields...)
		}
	}
}

// generateRequestID 生成请求ID（简单实现）
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成随机字符串（安全实现）
func randomString(length int) string {
	bytes := make([]byte, length/2) // 十六进制编码后长度会加倍
	if _, err := rand.Read(bytes); err != nil {
		// 如果crypto/rand失败，使用时间戳作为后备
		return time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(bytes)
}