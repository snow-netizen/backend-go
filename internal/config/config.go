package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	cfg  *Config
	once sync.Once
)

// Config 应用配置结构
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Database DatabaseConfig `mapstructure:"database"`
	Security SecurityConfig `mapstructure:"security"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Superuser SuperuserConfig `mapstructure:"superuser"`
	Log      LogConfig      `mapstructure:"log"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name         string `mapstructure:"name"`
	Version      string `mapstructure:"version"`
	Env          string `mapstructure:"env"`
	Port         int    `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret              string `mapstructure:"secret"`
	Issuer              string `mapstructure:"issuer"`
	Audience            string `mapstructure:"audience"`
	AccessTokenExpiry   int    `mapstructure:"access_token_expiry"`
	RefreshTokenExpiry  int    `mapstructure:"refresh_token_expiry"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	DSN             string `mapstructure:"dsn"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	PasswordCost int `mapstructure:"password_cost"`
	Captcha      struct {
		Width  int `mapstructure:"width"`
		Height int `mapstructure:"height"`
		Length int `mapstructure:"length"`
		Expiry int `mapstructure:"expiry"`
	} `mapstructure:"captcha"`
	Login struct {
		MaxFailures  int `mapstructure:"max_failures"`
		LockDuration int `mapstructure:"lock_duration"`
	} `mapstructure:"login"`
	RateLimit struct {
		Enabled           bool `mapstructure:"enabled"`
		RequestsPerSecond int  `mapstructure:"requests_per_second"`
		Burst             int  `mapstructure:"burst"`
	} `mapstructure:"rate_limit"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

// SuperuserConfig 超级管理员配置
type SuperuserConfig struct {
	Username string `mapstructure:"username"`
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level               string `mapstructure:"level"`                  // debug, info, warn, error
	Format              string `mapstructure:"format"`                 // console或json
	OutputPath          string `mapstructure:"output_path"`            // 文件路径，空则输出到stdout
	MaxSize             int    `mapstructure:"max_size"`               // 日志文件轮转大小(MB)
	MaxBackups          int    `mapstructure:"max_backups"`            // 保留备份数
	MaxAge              int    `mapstructure:"max_age"`                // 保留天数
	EnableDBLog         bool   `mapstructure:"enable_db_log"`          // 是否启用详细SQL日志
	SlowQueryThreshold  int    `mapstructure:"slow_query_threshold"`   // 慢查询阈值(ms)
}

// Load 加载配置
func Load() (*Config, error) {
	var err error
	once.Do(func() {
		cfg, err = loadConfig()
	})
	return cfg, err
}

// Get 获取配置实例
func Get() *Config {
	if cfg == nil {
		panic("config not loaded")
	}
	return cfg
}

// loadConfig 从YAML文件加载配置
func loadConfig() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	
	// 设置默认值
	setDefaults(v)
	
	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// 绑定环境变量
	bindEnvVars(v)
	
	// 解析配置
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Configuration loaded successfully",
		zap.String("app", config.App.Name),
		zap.String("env", config.App.Env),
		zap.Int("port", config.App.Port),
	)
	
	return &config, nil
}

// setDefaults 设置配置默认值
func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "admin-system-go")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.read_timeout", 30)
	v.SetDefault("app.write_timeout", 30)
	
	v.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	v.SetDefault("jwt.issuer", "admin-system-go")
	v.SetDefault("jwt.audience", "admin-system-users")
	v.SetDefault("jwt.access_token_expiry", 3600)
	v.SetDefault("jwt.refresh_token_expiry", 604800)
	
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "admin_system.db")
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_idle_conns", 20)
	v.SetDefault("database.conn_max_lifetime", 300)
	
	v.SetDefault("security.password_cost", 12)
	v.SetDefault("security.captcha.width", 120)
	v.SetDefault("security.captcha.height", 40)
	v.SetDefault("security.captcha.length", 6)
	v.SetDefault("security.captcha.expiry", 300)
	v.SetDefault("security.login.max_failures", 5)
	v.SetDefault("security.login.lock_duration", 900)
	v.SetDefault("security.rate_limit.enabled", true)
	v.SetDefault("security.rate_limit.requests_per_second", 10)
	v.SetDefault("security.rate_limit.burst", 30)
	
	v.SetDefault("cors.allowed_origins", []string{"http://localhost:3000", "http://localhost:5173"})
	v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"})
	v.SetDefault("cors.allowed_headers", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"})
	v.SetDefault("cors.allow_credentials", true)
	v.SetDefault("cors.max_age", 86400)
	
	v.SetDefault("superuser.username", "admin")
	v.SetDefault("superuser.email", "admin@example.com")
	v.SetDefault("superuser.password", "admin123")

	// 日志配置默认值
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output_path", "")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 7)
	v.SetDefault("log.max_age", 30)
	v.SetDefault("log.enable_db_log", false)
	v.SetDefault("log.slow_query_threshold", 200)
}

// bindEnvVars 绑定环境变量
func bindEnvVars(v *viper.Viper) {
	// 数据库环境变量
	v.BindEnv("database.dsn", "DATABASE_URL")
	v.BindEnv("database.driver", "DB_DRIVER")
	
	// JWT环境变量
	v.BindEnv("jwt.secret", "JWT_SECRET")
	
	// 应用环境变量
	v.BindEnv("app.env", "APP_ENV")
	v.BindEnv("app.port", "PORT")
	
	// CORS环境变量
	v.BindEnv("cors.allowed_origins", "CORS_ORIGINS")

	// 日志环境变量
	v.BindEnv("log.level", "LOG_LEVEL")
	v.BindEnv("log.format", "LOG_FORMAT")
	v.BindEnv("log.output_path", "LOG_OUTPUT_PATH")
	v.BindEnv("log.enable_db_log", "LOG_ENABLE_DB_LOG")
}

// validateConfig 验证配置
func validateConfig(cfg *Config) error {
	if cfg.App.Port < 1 || cfg.App.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.App.Port)
	}
	
	if cfg.JWT.Secret == "" || cfg.JWT.Secret == "your-secret-key-change-in-production" {
		return fmt.Errorf("JWT secret must be set")
	}
	
	if cfg.Database.Driver != "sqlite" && cfg.Database.Driver != "postgres" && cfg.Database.Driver != "mysql" {
		return fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}
	
	if cfg.Security.PasswordCost < 4 || cfg.Security.PasswordCost > 31 {
		return fmt.Errorf("password cost must be between 4 and 31")
	}
	
	return nil
}

// GetReadTimeout 获取读取超时时间
func (c *AppConfig) GetReadTimeout() time.Duration {
	return time.Duration(c.ReadTimeout) * time.Second
}

// GetWriteTimeout 获取写入超时时间
func (c *AppConfig) GetWriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeout) * time.Second
}

// GetAccessTokenExpiry 获取访问令牌过期时间
func (c *JWTConfig) GetAccessTokenExpiry() time.Duration {
	return time.Duration(c.AccessTokenExpiry) * time.Second
}

// GetRefreshTokenExpiry 获取刷新令牌过期时间
func (c *JWTConfig) GetRefreshTokenExpiry() time.Duration {
	return time.Duration(c.RefreshTokenExpiry) * time.Second
}