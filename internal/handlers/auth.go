package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"admin-system-go/internal/config"
	"admin-system-go/internal/logger"
	"admin-system-go/internal/models"
	"admin-system-go/internal/repositories"
	"admin-system-go/internal/security"
	"admin-system-go/pkg/response"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	userRepo    repositories.UserRepository
	captchaRepo repositories.CaptchaRepository
	jwtManager  *security.JWTManager
	hasher      security.PasswordHasher
	config      *config.Config
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(db *gorm.DB, jwtManager *security.JWTManager, hasher security.PasswordHasher) *AuthHandler {
	return &AuthHandler{
		userRepo:    repositories.NewUserRepository(db),
		captchaRepo: repositories.NewCaptchaRepository(db),
		jwtManager:  jwtManager,
		hasher:      hasher,
		config:      config.Get(),
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string       `json:"token"`
	TokenType string       `json:"token_type"`
	ExpiresIn int64        `json:"expires_in"`
	User      *models.User `json:"user"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name,omitempty"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// Login 用户登录
// @Summary 用户登录
// @Description 使用用户名和密码登录系统
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录信息"
// @Success 200 {object} response.Response{data=LoginResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 记录登录尝试
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	logger.Info("Login attempt",
		zap.String("username", req.Username),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", userAgent),
	)

	// 获取用户
	user, err := h.userRepo.GetByUsername(req.Username)
	if err != nil {
		logger.Error("Failed to query user",
			zap.String("username", req.Username),
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to query user")
		return
	}

	// 用户不存在
	if user == nil {
		logger.Warn("User not found",
			zap.String("username", req.Username),
			zap.String("client_ip", clientIP),
		)
		response.Unauthorized(c, "Invalid username or password")
		return
	}

	// 检查用户是否被锁定
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		logger.Warn("Account locked",
			zap.String("username", req.Username),
			zap.Uint("user_id", user.ID),
			zap.String("client_ip", clientIP),
			zap.Time("locked_until", *user.LockedUntil),
		)
		response.Forbidden(c, "Account is locked")
		return
	}

	// 验证密码
	if !h.hasher.Verify(user.PasswordHash, req.Password) {
		// 增加登录失败次数
		h.userRepo.IncrementLoginFailures(user.ID)

		// 检查是否达到最大失败次数
		if user.LoginFailures+1 >= h.config.Security.Login.MaxFailures {
			lockUntil := time.Now().Add(time.Duration(h.config.Security.Login.LockDuration) * time.Second)
			h.userRepo.LockUser(user.ID, lockUntil)
			logger.Warn("Account locked due to too many failed login attempts",
				zap.String("username", req.Username),
				zap.Uint("user_id", user.ID),
				zap.String("client_ip", clientIP),
				zap.Int("login_failures", user.LoginFailures+1),
				zap.Int("max_failures", h.config.Security.Login.MaxFailures),
				zap.Time("locked_until", lockUntil),
			)
			response.Forbidden(c, "Account has been locked due to too many failed login attempts")
			return
		}

		logger.Warn("Invalid password",
			zap.String("username", req.Username),
			zap.Uint("user_id", user.ID),
			zap.String("client_ip", clientIP),
			zap.Int("login_failures", user.LoginFailures+1),
			zap.Int("max_failures", h.config.Security.Login.MaxFailures),
		)
		response.Unauthorized(c, "Invalid username or password")
		return
	}

	// 检查用户是否启用
	if !user.IsActive {
		logger.Warn("Account disabled",
			zap.String("username", req.Username),
			zap.Uint("user_id", user.ID),
			zap.String("client_ip", clientIP),
		)
		response.Forbidden(c, "Account is disabled")
		return
	}

	// 重置登录失败次数和锁定状态
	h.userRepo.ResetLoginFailures(user.ID)
	h.userRepo.UnlockUser(user.ID)

	// 更新最后登录时间
	h.userRepo.UpdateLastLogin(user.ID)

	// 生成JWT token
	token, err := h.jwtManager.GenerateAccessToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		logger.Error("Failed to generate token",
			zap.String("username", req.Username),
			zap.Uint("user_id", user.ID),
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to generate token")
		return
	}

	// 返回响应
	loginResp := LoginResponse{
		Token:     token,
		TokenType: "bearer",
		ExpiresIn: int64(h.config.JWT.AccessTokenExpiry),
		User:      user,
	}

	logger.Info("Login successful",
		zap.String("username", req.Username),
		zap.Uint("user_id", user.ID),
		zap.String("client_ip", clientIP),
		zap.String("role", string(user.Role)),
		zap.Int64("token_expiry", int64(h.config.JWT.AccessTokenExpiry)),
	)

	response.Success(c, loginResp)
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册信息"
// @Success 200 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 记录注册尝试
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	logger.Info("Registration attempt",
		zap.String("username", req.Username),
		zap.String("email", req.Email),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", userAgent),
	)

	// 检查用户名是否已存在
	existingUser, err := h.userRepo.GetByUsername(req.Username)
	if err != nil {
		response.InternalServerError(c, "Failed to check username")
		return
	}
	if existingUser != nil {
		logger.Warn("Registration failed: username already exists",
			zap.String("username", req.Username),
			zap.String("client_ip", clientIP),
		)
		response.BadRequest(c, "Username already exists")
		return
	}

	// 检查邮箱是否已存在
	existingUser, err = h.userRepo.GetByEmail(req.Email)
	if err != nil {
		response.InternalServerError(c, "Failed to check email")
		return
	}
	if existingUser != nil {
		logger.Warn("Registration failed: email already exists",
			zap.String("email", req.Email),
			zap.String("client_ip", clientIP),
		)
		response.BadRequest(c, "Email already exists")
		return
	}

	// 生成密码哈希
	passwordHash, err := h.hasher.Hash(req.Password)
	if err != nil {
		logger.Error("Registration failed: failed to hash password",
			zap.String("username", req.Username),
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to hash password")
		return
	}

	// 创建用户
	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		FullName:     req.FullName,
		IsActive:     true,
		Role:         models.RoleUser,
	}

	if err := h.userRepo.Create(user); err != nil {
		logger.Error("Registration failed: failed to create user",
			zap.String("username", req.Username),
			zap.String("email", req.Email),
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to create user")
		return
	}

	// 不返回密码哈希
	user.PasswordHash = ""

	logger.Info("Registration successful",
		zap.String("username", req.Username),
		zap.String("email", req.Email),
		zap.String("client_ip", clientIP),
		zap.Uint("user_id", user.ID),
		zap.String("role", string(user.Role)),
	)
	response.Created(c, user)
}

// GetProfile 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=models.User}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	userID, exists := c.Get("user_id")
	if !exists {
		logger.Warn("GetProfile failed: user not authenticated",
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		)
		response.Unauthorized(c, "User not authenticated")
		return
	}

	uid := userID.(uint)
	user, err := h.userRepo.GetByID(uid)
	if err != nil {
		logger.Error("GetProfile failed: failed to get user",
			zap.Uint("user_id", uid),
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to get user")
		return
	}

	if user == nil {
		logger.Warn("GetProfile failed: user not found",
			zap.Uint("user_id", uid),
			zap.String("client_ip", clientIP),
		)
		response.NotFound(c, "User not found")
		return
	}

	// 不返回密码哈希
	user.PasswordHash = ""
	response.Success(c, user)
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前登录用户的密码
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "密码修改信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	userID, exists := c.Get("user_id")
	if !exists {
		logger.Warn("ChangePassword failed: user not authenticated",
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		)
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 获取用户
	uid := userID.(uint)
	user, err := h.userRepo.GetByID(uid)
	if err != nil {
		logger.Error("ChangePassword failed: failed to get user",
			zap.Uint("user_id", uid),
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to get user")
		return
	}

	if user == nil {
		logger.Warn("ChangePassword failed: user not found",
			zap.Uint("user_id", uid),
			zap.String("client_ip", clientIP),
		)
		response.NotFound(c, "User not found")
		return
	}

	// 验证旧密码
	if !h.hasher.Verify(user.PasswordHash, req.OldPassword) {
		logger.Warn("ChangePassword failed: old password incorrect",
			zap.Uint("user_id", uid),
			zap.String("client_ip", clientIP),
			zap.String("username", user.Username),
		)
		response.BadRequest(c, "Old password is incorrect")
		return
	}

	// 生成新密码哈希
	newPasswordHash, err := h.hasher.Hash(req.NewPassword)
	if err != nil {
		logger.Error("ChangePassword failed: failed to hash new password",
			zap.Uint("user_id", uid),
			zap.String("client_ip", clientIP),
			zap.String("username", user.Username),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to hash new password")
		return
	}

	// 更新密码
	if err := h.userRepo.ChangePassword(user.ID, newPasswordHash); err != nil {
		logger.Error("ChangePassword failed: failed to update password in database",
			zap.Uint("user_id", uid),
			zap.String("client_ip", clientIP),
			zap.String("username", user.Username),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to change password")
		return
	}

	logger.Info("Password changed successfully",
		zap.Uint("user_id", uid),
		zap.String("client_ip", clientIP),
		zap.String("username", user.Username),
	)
	response.SuccessWithMessage(c, "Password changed successfully", nil)
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 对于JWT，登出只需要前端删除token
	// 如果需要服务端控制，可以在这里将token加入黑名单（需要Redis）
	response.SuccessWithMessage(c, "Logout successful", nil)
}

// GenerateCaptcha 生成验证码
// @Summary 生成验证码
// @Description 生成验证码图片，返回Base64编码的图片数据
// @Tags 认证
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /auth/captcha [get]
func (h *AuthHandler) GenerateCaptcha(c *gin.Context) {
	// 清理过期的验证码
	if err := h.captchaRepo.DeleteExpired(); err != nil {
		response.InternalServerError(c, "Failed to clean expired captchas")
		return
	}

	// 生成验证码图片
	captchaImage, err := security.GenerateCaptchaImage(
		h.config.Security.Captcha.Width,
		h.config.Security.Captcha.Height,
		h.config.Security.Captcha.Length,
	)
	if err != nil {
		response.InternalServerError(c, "Failed to generate captcha image")
		return
	}

	// 保存验证码到数据库
	captchaRecord := &models.CaptchaCode{
		CodeID:    captchaImage.ID,
		Code:      captchaImage.Code,
		Content:   captchaImage.Image,
		ExpiresAt: captchaImage.Expires,
		Used:      false,
	}

	if err := h.captchaRepo.Save(captchaRecord); err != nil {
		response.InternalServerError(c, "Failed to save captcha")
		return
	}

	// 返回验证码信息
	captchaResp := map[string]interface{}{
		"captcha_id": captchaImage.ID,
		"image":      captchaImage.Image,
		"expires_in": h.config.Security.Captcha.Expiry,
	}

	response.Success(c, captchaResp)
}

// VerifyCaptcha 验证验证码
func (h *AuthHandler) VerifyCaptcha(captchaID, code string) bool {
	// 验证验证码
	valid, err := h.captchaRepo.Verify(captchaID, code)
	if err != nil {
		return false
	}

	if !valid {
		return false
	}

	// 标记验证码为已使用
	if err := h.captchaRepo.MarkAsUsed(captchaID); err != nil {
		return false
	}

	return true
}

// RegisterRoutes 注册路由
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/register", h.Register)
		auth.GET("/captcha", h.GenerateCaptcha)

		// 需要认证的路由
		auth.GET("/profile", h.GetProfile)
		auth.POST("/logout", h.Logout)
		auth.POST("/change-password", h.ChangePassword)
	}
}
