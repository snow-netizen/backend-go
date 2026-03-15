package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"admin-system-go/internal/logger"
	"admin-system-go/internal/security"
	"admin-system-go/pkg/response"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	jwtManager *security.JWTManager
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(jwtManager *security.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager}
}

// RequireAuth 需要认证的中间件
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求信息用于日志
		clientIP := c.ClientIP()
		requestMethod := c.Request.Method
		requestPath := c.Request.URL.Path

		// 从请求头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Authorization header missing",
				zap.String("client_ip", clientIP),
				zap.String("method", requestMethod),
				zap.String("path", requestPath),
			)
			response.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}
		
		// 检查Bearer token格式
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logger.Warn("Invalid authorization header format",
				zap.String("client_ip", clientIP),
				zap.String("method", requestMethod),
				zap.String("path", requestPath),
				zap.Int("header_length", len(authHeader)),
			)
			response.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}
		
		token := parts[1]
		
		// 验证token
		claims, err := m.jwtManager.VerifyToken(token)
		if err != nil {
			logger.Warn("Token verification failed",
				zap.String("client_ip", clientIP),
				zap.String("method", requestMethod),
				zap.String("path", requestPath),
				zap.Error(err),
				zap.Int("token_length", len(token)),
			)
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}
		
		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		logger.Info("Token verification successful",
			zap.String("client_ip", clientIP),
			zap.String("method", requestMethod),
			zap.String("path", requestPath),
			zap.Uint("user_id", claims.UserID),
			zap.String("username", claims.Username),
			zap.String("role", claims.Role),
		)

		c.Next()
	}
}

// RequireRole 需要特定角色的中间件
func (m *AuthMiddleware) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先进行认证
		m.RequireAuth()(c)
		if c.IsAborted() {
			return
		}

		// 获取请求和用户信息用于日志
		clientIP := c.ClientIP()
		requestMethod := c.Request.Method
		requestPath := c.Request.URL.Path
		userIDVal, _ := c.Get("user_id")
		usernameVal, _ := c.Get("username")

		var userID uint
		var username string

		if uid, ok := userIDVal.(uint); ok {
			userID = uid
		}
		if uname, ok := usernameVal.(string); ok {
			username = uname
		}

		// 检查角色
		role, exists := c.Get("role")
		if !exists {
			logger.Warn("User role not found in context",
				zap.String("client_ip", clientIP),
				zap.String("method", requestMethod),
				zap.String("path", requestPath),
				zap.Uint("user_id", userID),
				zap.String("username", username),
				zap.String("required_role", requiredRole),
			)
			response.Forbidden(c, "User role not found")
			c.Abort()
			return
		}
		
		roleStr, ok := role.(string)
		if !ok {
			logger.Warn("Role type assertion failed",
				zap.String("client_ip", clientIP),
				zap.String("method", requestMethod),
				zap.String("path", requestPath),
				zap.Uint("user_id", userID),
				zap.String("username", username),
				zap.String("required_role", requiredRole),
				zap.Any("actual_role", role),
			)
			response.Forbidden(c, "User role not found")
			c.Abort()
			return
		}

		if roleStr != requiredRole {
			logger.Warn("Insufficient permissions",
				zap.String("client_ip", clientIP),
				zap.String("method", requestMethod),
				zap.String("path", requestPath),
				zap.Uint("user_id", userID),
				zap.String("username", username),
				zap.String("actual_role", roleStr),
				zap.String("required_role", requiredRole),
			)
			response.Forbidden(c, "Insufficient permissions")
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// RequireAdmin 需要管理员角色的中间件
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole("admin")
}

// GetUserIDFromContext 从上下文中获取用户ID
func GetUserIDFromContext(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	
	// 类型断言
	id, ok := userID.(uint)
	if !ok {
		// 尝试从float64转换（JSON数字默认是float64）
		if floatID, ok := userID.(float64); ok {
			return uint(floatID), true
		}
		return 0, false
	}
	
	return id, true
}

// GetUsernameFromContext 从上下文中获取用户名
func GetUsernameFromContext(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	
	str, ok := username.(string)
	return str, ok
}

// GetRoleFromContext 从上下文中获取角色
func GetRoleFromContext(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	
	str, ok := role.(string)
	return str, ok
}