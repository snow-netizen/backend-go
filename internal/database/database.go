package database

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"admin-system-go/internal/config"
	appLogger "admin-system-go/internal/logger"
	"admin-system-go/internal/models"
)

var (
	db     *gorm.DB
	dbOnce sync.Once
)

// DB 获取数据库连接实例
func DB() *gorm.DB {
	if db == nil {
		panic("database not initialized")
	}
	return db
}

// Init 初始化数据库连接
func Init() error {
	var err error
	
	dbOnce.Do(func() {
		cfg := config.Get()
		
		// 配置GORM日志（使用Zap适配器）
		gormLogger := appLogger.NewGormLogger(appLogger.Get(), &cfg.Log)
		
		// 根据驱动创建数据库连接
		switch cfg.Database.Driver {
		case "postgres":
			db, err = gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{
				Logger: gormLogger,
			})
		case "mysql":
			// 如果需要mysql支持，可以添加
			err = fmt.Errorf("mysql driver not implemented yet")
		case "sqlite":
			db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{
				Logger: gormLogger,
			})
		default:
			err = fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
		}
		
		if err != nil {
			err = fmt.Errorf("failed to connect to database: %w", err)
			return
		}
		
		// 获取通用数据库对象 sql.DB
		sqlDB, err := db.DB()
		if err != nil {
			err = fmt.Errorf("failed to get sql.DB: %w", err)
			return
		}
		
		// 配置连接池
		sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)
		
		// 测试连接
		if err := sqlDB.Ping(); err != nil {
			err = fmt.Errorf("failed to ping database: %w", err)
			return
		}
		
		appLogger.Info("Database connection established successfully",
			zap.String("driver", cfg.Database.Driver),
			zap.Int("max_open_conns", cfg.Database.MaxOpenConns),
			zap.Int("max_idle_conns", cfg.Database.MaxIdleConns),
			zap.Int("conn_max_lifetime", cfg.Database.ConnMaxLifetime),
		)
	})
	
	return err
}

// Close 关闭数据库连接
func Close() error {
	if db == nil {
		return nil
	}
	
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	
	return sqlDB.Close()
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	// 执行迁移
	if err := db.AutoMigrate(
		&models.User{},
		&models.CaptchaCode{},
		&models.LoginLog{},
		&models.OperationLog{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	
	appLogger.Info("Database migration completed",
		zap.String("models", "User, CaptchaCode, LoginLog, OperationLog"),
	)
	return nil
}

// Transaction 执行数据库事务
func Transaction(fc func(tx *gorm.DB) error) error {
	return db.Transaction(fc)
}

// WithTx 获取事务上下文
func WithTx(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return db
}

// HealthCheck 数据库健康检查
func HealthCheck() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	
	return sqlDB.Ping()
}

// Stats 获取数据库连接统计信息
func Stats() map[string]interface{} {
	if db == nil {
		return nil
	}
	
	sqlDB, err := db.DB()
	if err != nil {
		return nil
	}
	
	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections":     stats.MaxOpenConnections,
		"open_connections":         stats.OpenConnections,
		"in_use":                   stats.InUse,
		"idle":                     stats.Idle,
		"wait_count":               stats.WaitCount,
		"wait_duration":            stats.WaitDuration.String(),
		"max_idle_closed":          stats.MaxIdleClosed,
		"max_lifetime_closed":      stats.MaxLifetimeClosed,
	}
}