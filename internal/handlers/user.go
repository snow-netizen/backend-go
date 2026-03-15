package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"admin-system-go/internal/logger"
	"admin-system-go/internal/models"
	"admin-system-go/internal/repositories"
	"admin-system-go/internal/security"
	"admin-system-go/pkg/response"
)

// UserHandler 用户处理器
type UserHandler struct {
	userRepo repositories.UserRepository
	hasher   security.PasswordHasher
}

// NewUserHandler 创建用户处理器
func NewUserHandler(db *gorm.DB, hasher security.PasswordHasher) *UserHandler {
	return &UserHandler{
		userRepo: repositories.NewUserRepository(db),
		hasher:   hasher,
	}
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string          `json:"username" binding:"required,min=3,max=50"`
	Email    string          `json:"email" binding:"required,email"`
	Password string          `json:"password" binding:"required,min=6"`
	FullName string          `json:"full_name,omitempty"`
	Role     models.UserRole `json:"role" binding:"required,oneof=user admin"`
	IsActive bool            `json:"is_active"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email    *string          `json:"email,omitempty" binding:"omitempty,email"`
	FullName *string          `json:"full_name,omitempty"`
	Role     *models.UserRole `json:"role,omitempty" binding:"omitempty,oneof=user admin"`
	IsActive *bool            `json:"is_active,omitempty"`
}

// ChangePasswordRequest 管理员修改用户密码请求
type AdminChangePasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表，支持分页、搜索和筛选
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param search query string false "搜索关键词"
// @Param role query string false "角色筛选"
// @Param is_active query bool false "状态筛选"
// @Success 200 {object} response.PageResponse{data=[]models.User}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	clientIP := c.ClientIP()
	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	search := c.Query("search")
	role := models.UserRole(c.Query("role"))
	isActiveStr := c.Query("is_active")

	// 记录查询参数
	logger.Info("ListUsers query",
		zap.String("client_ip", clientIP),
		zap.Int("page", page),
		zap.Int("size", size),
		zap.String("search", search),
		zap.String("role", string(role)),
		zap.String("is_active", isActiveStr),
	)

	// 构建筛选条件
	filter := repositories.UserFilter{
		Search: search,
		Role:   role,
	}

	if isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	// 获取用户列表
	users, total, err := h.userRepo.List(filter, page, size)
	if err != nil {
		logger.Error("ListUsers failed: failed to list users",
			zap.String("client_ip", clientIP),
			zap.Error(err),
		)
		response.InternalServerError(c, "Failed to list users")
		return
	}

	// 移除密码哈希
	for i := range users {
		users[i].PasswordHash = ""
	}

	logger.Info("ListUsers successful",
		zap.String("client_ip", clientIP),
		zap.Int("page", page),
		zap.Int("size", size),
		zap.Int64("total", total),
		zap.Int("count", len(users)),
	)
	response.SuccessPage(c, users, page, size, total)
}

// GetUser 获取用户详情
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		response.InternalServerError(c, "Failed to get user")
		return
	}

	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	// 移除密码哈希
	user.PasswordHash = ""
	response.Success(c, user)
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户账号
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserRequest true "用户信息"
// @Success 201 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 检查用户名是否已存在
	existingUser, err := h.userRepo.GetByUsername(req.Username)
	if err != nil {
		response.InternalServerError(c, "Failed to check username")
		return
	}
	if existingUser != nil {
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
		response.BadRequest(c, "Email already exists")
		return
	}

	// 生成密码哈希
	passwordHash, err := h.hasher.Hash(req.Password)
	if err != nil {
		response.InternalServerError(c, "Failed to hash password")
		return
	}

	// 创建用户
	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		FullName:     req.FullName,
		IsActive:     req.IsActive,
		Role:         req.Role,
	}

	if err := h.userRepo.Create(user); err != nil {
		response.InternalServerError(c, "Failed to create user")
		return
	}

	// 移除密码哈希
	user.PasswordHash = ""
	response.Created(c, user)
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param request body UpdateUserRequest true "更新信息"
// @Success 200 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// 检查用户是否存在
	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		response.InternalServerError(c, "Failed to get user")
		return
	}
	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 更新用户字段
	if req.Email != nil && *req.Email != user.Email {
		// 检查邮箱是否被其他用户使用
		existingUser, err := h.userRepo.GetByEmail(*req.Email)
		if err != nil {
			response.InternalServerError(c, "Failed to check email")
			return
		}
		if existingUser != nil && existingUser.ID != user.ID {
			response.BadRequest(c, "Email already used by another user")
			return
		}
		user.Email = *req.Email
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}

	if req.Role != nil {
		user.Role = *req.Role
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	// 保存更新
	if err := h.userRepo.Update(user); err != nil {
		response.InternalServerError(c, "Failed to update user")
		return
	}

	// 移除密码哈希
	user.PasswordHash = ""
	response.Success(c, user)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除指定用户
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if exists && uint(id) == currentUserID.(uint) {
		response.BadRequest(c, "Cannot delete yourself")
		return
	}

	// 检查用户是否存在
	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		response.InternalServerError(c, "Failed to get user")
		return
	}
	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	// 删除用户
	if err := h.userRepo.Delete(uint(id)); err != nil {
		response.InternalServerError(c, "Failed to delete user")
		return
	}

	response.SuccessWithMessage(c, "User deleted successfully", nil)
}

// ChangeUserPassword 管理员修改用户密码
// @Summary 管理员修改用户密码
// @Description 管理员修改指定用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Param request body AdminChangePasswordRequest true "新密码"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id}/password [put]
func (h *UserHandler) ChangeUserPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// 检查用户是否存在
	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		response.InternalServerError(c, "Failed to get user")
		return
	}
	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	var req AdminChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 生成新密码哈希
	newPasswordHash, err := h.hasher.Hash(req.NewPassword)
	if err != nil {
		response.InternalServerError(c, "Failed to hash new password")
		return
	}

	// 更新密码
	if err := h.userRepo.ChangePassword(user.ID, newPasswordHash); err != nil {
		response.InternalServerError(c, "Failed to change password")
		return
	}

	response.SuccessWithMessage(c, "Password changed successfully", nil)
}

// ChangeUserStatus 修改用户状态
// @Summary 修改用户状态
// @Description 启用或禁用用户账号
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users/{id}/status [put]
func (h *UserHandler) ChangeUserStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// 检查用户是否存在
	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		response.InternalServerError(c, "Failed to get user")
		return
	}
	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if exists && uint(id) == currentUserID.(uint) {
		response.BadRequest(c, "Cannot change your own status")
		return
	}

	// 切换状态
	user.IsActive = !user.IsActive

	// 保存更新
	if err := h.userRepo.Update(user); err != nil {
		response.InternalServerError(c, "Failed to update user status")
		return
	}

	status := "enabled"
	if !user.IsActive {
		status = "disabled"
	}

	response.SuccessWithMessage(c, "User status changed to "+status, user)
}

// RegisterRoutes 注册路由
func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup) {
	users := router.Group("/users")
	{
		users.GET("/", h.ListUsers)
		users.POST("/", h.CreateUser)
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
		users.PUT("/:id/password", h.ChangeUserPassword)
		users.PUT("/:id/status", h.ChangeUserStatus)
	}
}
