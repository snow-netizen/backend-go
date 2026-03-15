package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"admin-system-go/internal/config"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
)

// Init 初始化全局logger
func Init(cfg *config.LogConfig) error {
	var err error
	once.Do(func() {
		globalLogger, err = createLogger(cfg)
	})
	return err
}

// Get 获取全局logger实例
func Get() *zap.Logger {
	if globalLogger == nil {
		// 如果logger未初始化，返回一个临时的生产环境logger
		logger, _ := zap.NewProduction()
		return logger
	}
	return globalLogger
}

// Sync 同步日志缓冲区
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Debug 记录Debug级别日志
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info 记录Info级别日志
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn 记录Warn级别日志
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error 记录Error级别日志
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// With 创建带字段的logger
func With(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}

// createLogger 根据配置创建logger
func createLogger(cfg *config.LogConfig) (*zap.Logger, error) {
	// 设置日志级别
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 编码器配置
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "console" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	}

	// 设置日志输出
	var writeSyncer zapcore.WriteSyncer
	if cfg.OutputPath != "" {
		// 文件输出，支持轮转
		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.OutputPath,
			MaxSize:    cfg.MaxSize,    // MB
			MaxBackups: cfg.MaxBackups, // 保留的旧文件数量
			MaxAge:     cfg.MaxAge,     // 保留天数
			Compress:   true,           // 压缩旧文件
		}
		writeSyncer = zapcore.AddSync(lumberjackLogger)
	} else {
		// 控制台输出
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 创建核心
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		writeSyncer,
		level,
	)

	// 创建logger
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	// 添加全局字段
	appCfg := config.Get()
	logger = logger.With(
		zap.String("app", appCfg.App.Name),
		zap.String("version", appCfg.App.Version),
		zap.String("env", appCfg.App.Env),
	)

	return logger, nil
}