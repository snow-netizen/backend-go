package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"admin-system-go/internal/config"
)

// GormLogger GORM日志适配器
type GormLogger struct {
	ZapLogger                *zap.Logger
	LogLevel                 logger.LogLevel
	SlowThreshold            time.Duration
	SkipCallerLookup         bool
	IgnoreRecordNotFoundError bool
	ParameterizedQueries     bool
}

// NewGormLogger 创建GORM日志适配器
func NewGormLogger(zapLogger *zap.Logger, cfg *config.LogConfig) logger.Interface {
	var logLevel logger.LogLevel
	if cfg.EnableDBLog {
		logLevel = logger.Info
	} else {
		logLevel = logger.Warn
	}

	return &GormLogger{
		ZapLogger:                zapLogger,
		LogLevel:                 logLevel,
		SlowThreshold:            time.Duration(cfg.SlowQueryThreshold) * time.Millisecond,
		SkipCallerLookup:         false,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:     false,
	}
}

// LogMode 设置日志级别
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info 打印Info级别日志
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.ZapLogger.Info(msg, zap.Any("data", data))
	}
}

// Warn 打印Warn级别日志
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.ZapLogger.Warn(msg, zap.Any("data", data))
	}
}

// Error 打印Error级别日志
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.ZapLogger.Error(msg, zap.Any("data", data))
	}
}

// Trace 打印SQL日志
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// 获取调用者信息
	file := utils.FileWithLineNum()
	if l.SkipCallerLookup {
		file = ""
	}

	// 记录错误
	if err != nil && !(l.IgnoreRecordNotFoundError && logger.ErrRecordNotFound == err) {
		l.ZapLogger.Error("database error",
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
			zap.String("file", file),
		)
		return
	}

	// 慢查询日志
	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold {
		l.ZapLogger.Warn("slow query detected",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
			zap.String("file", file),
			zap.Duration("threshold", l.SlowThreshold),
		)
		return
	}

	// 普通SQL日志
	if l.LogLevel >= logger.Info {
		l.ZapLogger.Debug("sql query",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
			zap.String("file", file),
		)
	}
}