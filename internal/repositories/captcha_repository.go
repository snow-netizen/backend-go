package repositories

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"admin-system-go/internal/models"
)

// CaptchaRepository 验证码仓库接口
type CaptchaRepository interface {
	Save(captcha *models.CaptchaCode) error
	GetByID(codeID string) (*models.CaptchaCode, error)
	Delete(codeID string) error
	DeleteExpired() error
	Verify(codeID, code string) (bool, error)
	MarkAsUsed(codeID string) error
}

// captchaRepository 验证码仓库实现
type captchaRepository struct {
	db *gorm.DB
}

// NewCaptchaRepository 创建验证码仓库
func NewCaptchaRepository(db *gorm.DB) CaptchaRepository {
	return &captchaRepository{db: db}
}

// Save 保存验证码
func (r *captchaRepository) Save(captcha *models.CaptchaCode) error {
	if err := r.db.Create(captcha).Error; err != nil {
		return fmt.Errorf("failed to save captcha: %w", err)
	}
	return nil
}

// GetByID 根据ID获取验证码
func (r *captchaRepository) GetByID(codeID string) (*models.CaptchaCode, error) {
	var captcha models.CaptchaCode
	if err := r.db.Where("code_id = ?", codeID).First(&captcha).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get captcha: %w", err)
	}
	return &captcha, nil
}

// Delete 删除验证码
func (r *captchaRepository) Delete(codeID string) error {
	if err := r.db.Where("code_id = ?", codeID).Delete(&models.CaptchaCode{}).Error; err != nil {
		return fmt.Errorf("failed to delete captcha: %w", err)
	}
	return nil
}

// DeleteExpired 删除所有过期的验证码
func (r *captchaRepository) DeleteExpired() error {
	if err := r.db.Where("expires_at < ?", time.Now()).Delete(&models.CaptchaCode{}).Error; err != nil {
		return fmt.Errorf("failed to delete expired captchas: %w", err)
	}
	return nil
}

// Verify 验证验证码
func (r *captchaRepository) Verify(codeID, code string) (bool, error) {
	captcha, err := r.GetByID(codeID)
	if err != nil {
		return false, err
	}

	if captcha == nil {
		return false, nil
	}

	// 检查是否已使用
	if captcha.Used {
		return false, nil
	}

	// 检查是否过期
	if captcha.ExpiresAt.Before(time.Now()) {
		return false, nil
	}

	// 验证码匹配
	return captcha.Code == code, nil
}

// MarkAsUsed 标记验证码为已使用
func (r *captchaRepository) MarkAsUsed(codeID string) error {
	if err := r.db.Model(&models.CaptchaCode{}).
		Where("code_id = ?", codeID).
		Update("used", true).Error; err != nil {
		return fmt.Errorf("failed to mark captcha as used: %w", err)
	}
	return nil
}
