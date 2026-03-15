package security

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"image/color"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mojocn/base64Captcha"
	"golang.org/x/crypto/bcrypt"

	"admin-system-go/internal/config"
)

// PasswordHasher 密码哈希接口
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hashedPassword, password string) bool
}

// bcryptHasher bcrypt密码哈希实现
type bcryptHasher struct {
	cost int
}

// NewBCryptHasher 创建bcrypt哈希器
func NewBCryptHasher(cost int) PasswordHasher {
	return &bcryptHasher{cost: cost}
}

// Hash 生成密码哈希
func (h *bcryptHasher) Hash(password string) (string, error) {
	// bcrypt限制密码长度为72字节
	if len(password) > 72 {
		password = password[:72]
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashed), nil
}

// Verify 验证密码
func (h *bcryptHasher) Verify(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// JWTManager JWT管理器
type JWTManager struct {
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewJWTManager 创建JWT管理器
func NewJWTManager() *JWTManager {
	cfg := config.Get()
	return &JWTManager{
		secret:     []byte(cfg.JWT.Secret),
		issuer:     cfg.JWT.Issuer,
		audience:   cfg.JWT.Audience,
		accessTTL:  cfg.JWT.GetAccessTokenExpiry(),
		refreshTTL: cfg.JWT.GetRefreshTokenExpiry(),
	}
}

// Claims JWT声明
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成访问令牌
func (m *JWTManager) GenerateAccessToken(userID uint, username, role string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    m.issuer,
			Audience:  []string{m.audience},
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// GenerateRefreshToken 生成刷新令牌
func (m *JWTManager) GenerateRefreshToken(userID uint) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    m.issuer,
		Audience:  []string{m.audience},
		Subject:   fmt.Sprintf("%d", userID),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// VerifyToken 验证JWT令牌
func (m *JWTManager) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateRandomCode 生成随机验证码
func GenerateRandomCode(length int) (string, error) {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, length)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random code: %w", err)
	}

	for i := range bytes {
		bytes[i] = charset[int(bytes[i])%len(charset)]
	}

	return string(bytes), nil
}

// CaptchaImage 验证码图片结构
type CaptchaImage struct {
	ID      string
	Code    string
	Image   string // Base64编码的图片
	Expires time.Time
}

// GenerateCaptchaImage 生成验证码图片
func GenerateCaptchaImage(width, height, codeLength int) (*CaptchaImage, error) {
	// 生成验证码ID
	id, err := GenerateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate captcha ID: %w", err)
	}

	// 配置验证码驱动
	driverString := base64Captcha.DriverString{
		Height:          height,
		Width:           width,
		NoiseCount:      0,                                 // 干扰线数量
		ShowLineOptions: base64Captcha.OptionShowSlimeLine, // 不显示横线
		// ShowLineOptions: base64Captcha.OptionShowSlimeLine | base64Captcha.OptionShowSineLine,
		Length: codeLength,
		Source: "1234567890qwertyuioplkjhgfdsazxcvbnm",
		BgColor: &color.RGBA{
			R: 240,
			G: 240,
			B: 240,
			A: 255,
		},
	}

	// 创建驱动
	driver := driverString.ConvertFonts()

	// 生成验证码
	captcha := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)

	// 生成验证码图片
	id, content, answer, err := captcha.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate captcha: %w", err)
	}

	// 构造Base64图片URL
	base64Image := content

	return &CaptchaImage{
		ID:      id,
		Code:    answer,
		Image:   base64Image,
		Expires: time.Now().Add(time.Duration(config.Get().Security.Captcha.Expiry) * time.Second),
	}, nil
}
