package repositories

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"admin-system-go/internal/models"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	GetByID(id uint) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	List(filter UserFilter, page, size int) ([]models.User, int64, error)
	Count(filter UserFilter) (int64, error)
	Create(user *models.User) error
	Update(user *models.User) error
	Delete(id uint) error
	UpdateLastLogin(userID uint) error
	IncrementLoginFailures(userID uint) error
	ResetLoginFailures(userID uint) error
	LockUser(userID uint, until time.Time) error
	UnlockUser(userID uint) error
	ChangePassword(userID uint, newPasswordHash string) error
}

// UserFilter 用户筛选条件
type UserFilter struct {
	Search   string
	Role     models.UserRole
	IsActive *bool
}

// userRepository 用户仓库实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// GetByID 根据ID获取用户
func (r *userRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// List 获取用户列表（支持搜索和筛选）
func (r *userRepository) List(filter UserFilter, page, size int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	
	query := r.db.Model(&models.User{})
	
	// 应用筛选条件
	r.applyFilter(query, filter)
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}
	
	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	
	return users, total, nil
}

// Count 统计用户数量
func (r *userRepository) Count(filter UserFilter) (int64, error) {
	var count int64
	
	query := r.db.Model(&models.User{})
	r.applyFilter(query, filter)
	
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	
	return count, nil
}

// applyFilter 应用筛选条件
func (r *userRepository) applyFilter(query *gorm.DB, filter UserFilter) {
	if filter.Search != "" {
		search := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ? OR LOWER(full_name) LIKE ?", 
			search, search, search)
	}
	
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
}

// Create 创建用户
func (r *userRepository) Create(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// Update 更新用户
func (r *userRepository) Update(user *models.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// Delete 删除用户
func (r *userRepository) Delete(id uint) error {
	if err := r.db.Delete(&models.User{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// UpdateLastLogin 更新最后登录时间
func (r *userRepository) UpdateLastLogin(userID uint) error {
	now := time.Now()
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at":   &now,
			"login_failures":  0,
			"locked_until":    nil,
		}).Error; err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// IncrementLoginFailures 增加登录失败次数
func (r *userRepository) IncrementLoginFailures(userID uint) error {
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).
		Update("login_failures", gorm.Expr("login_failures + 1")).Error; err != nil {
		return fmt.Errorf("failed to increment login failures: %w", err)
	}
	return nil
}

// ResetLoginFailures 重置登录失败次数
func (r *userRepository) ResetLoginFailures(userID uint) error {
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).
		Update("login_failures", 0).Error; err != nil {
		return fmt.Errorf("failed to reset login failures: %w", err)
	}
	return nil
}

// LockUser 锁定用户
func (r *userRepository) LockUser(userID uint, until time.Time) error {
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).
		Update("locked_until", until).Error; err != nil {
		return fmt.Errorf("failed to lock user: %w", err)
	}
	return nil
}

// UnlockUser 解锁用户
func (r *userRepository) UnlockUser(userID uint) error {
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"locked_until":    nil,
			"login_failures":  0,
		}).Error; err != nil {
		return fmt.Errorf("failed to unlock user: %w", err)
	}
	return nil
}

// ChangePassword 修改密码
func (r *userRepository) ChangePassword(userID uint, newPasswordHash string) error {
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).
		Update("password_hash", newPasswordHash).Error; err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}
	return nil
}