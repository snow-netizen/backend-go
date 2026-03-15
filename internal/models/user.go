package models

import (
	"time"

	"gorm.io/gorm"
)

// UserRole 用户角色枚举
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// User 用户模型
type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Username       string    `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email          string    `gorm:"size:100;uniqueIndex;not null" json:"email"`
	PasswordHash   string    `gorm:"size:255;not null" json:"-"`
	FullName       string    `gorm:"size:100" json:"full_name,omitempty"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	Role           UserRole  `gorm:"type:varchar(20);default:'user'" json:"role"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	LoginFailures  int       `gorm:"default:0" json:"-"`
	LockedUntil    *time.Time `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 返回表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate 创建前的钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate 更新前的钩子
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

// CaptchaCode 验证码模型
type CaptchaCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CodeID    string    `gorm:"size:64;uniqueIndex;not null" json:"code_id"`
	Code      string    `gorm:"size:10;not null" json:"-"`
	Content   string    `gorm:"size:255" json:"-"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 返回表名
func (CaptchaCode) TableName() string {
	return "captcha_codes"
}

// LoginLog 登录日志模型
type LoginLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Username  string    `gorm:"size:50;index;not null" json:"username"`
	LoginType string    `gorm:"size:20" json:"login_type"`
	IP        string    `gorm:"size:45" json:"ip"`
	UserAgent string    `gorm:"size:500" json:"user_agent"`
	Success   bool      `gorm:"default:false" json:"success"`
	FailReason string   `gorm:"size:200" json:"fail_reason,omitempty"`
	Location  string    `gorm:"size:100" json:"location,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 返回表名
func (LoginLog) TableName() string {
	return "login_logs"
}

// OperationLog 操作日志模型
type OperationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Username  string    `gorm:"size:50;index;not null" json:"username"`
	Action    string    `gorm:"size:100;not null" json:"action"`
	Module    string    `gorm:"size:50;not null" json:"module"`
	IP        string    `gorm:"size:45" json:"ip"`
	Params    string    `gorm:"type:text" json:"params,omitempty"`
	Result    string    `gorm:"type:text" json:"result,omitempty"`
	ErrorMsg  string    `gorm:"size:500" json:"error_msg,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 返回表名
func (OperationLog) TableName() string {
	return "operation_logs"
}