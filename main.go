package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	// 👇 必须加这一行！替换成你自己的项目模块名
	_ "admin-system-go/docs"
	"admin-system-go/internal/config"
	"admin-system-go/internal/database"
	"admin-system-go/internal/handlers"
	"admin-system-go/internal/logger"
	"admin-system-go/internal/middleware"
	"admin-system-go/internal/models"
	"admin-system-go/internal/repositories"
	"admin-system-go/internal/security"
)

// @title Admin System Go API
// @version 1.0
// @description 管理系统后端API - 基于Gin框架的RESTful API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8000
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @type apiKey
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志系统
	if err := logger.Init(&cfg.Log); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// 初始化数据库
	if err := database.Init(); err != nil {
		logger.Error("Failed to initialize database",
			zap.Error(err),
		)
		os.Exit(1)
	}
	defer database.Close()

	// 自动迁移数据库表
	if err := database.AutoMigrate(); err != nil {
		logger.Error("Failed to migrate database",
			zap.Error(err),
		)
		os.Exit(1)
	}

	// 创建初始超级管理员
	if err := createSuperUser(); err != nil {
		logger.Error("Failed to create super user",
			zap.Error(err),
		)
	}

	// 设置Gin模式
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 创建Gin路由
	router := gin.New()

	// 全局中间件
	router.Use(middleware.LoggingMiddleware())
	router.Use(gin.Recovery())

	// CORS配置
	corsConfig := cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     cfg.CORS.AllowedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           time.Duration(cfg.CORS.MaxAge) * time.Second,
	}
	router.Use(cors.New(corsConfig))

	// 健康检查路由
	router.GET("/health", healthCheck)
	router.GET("/", rootHandler)

	// Swagger文档路由
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 路由组
	apiV1 := router.Group("/api/v1")
	{
		// 初始化依赖
		db := database.DB()
		jwtManager := security.NewJWTManager()
		hasher := security.NewBCryptHasher(cfg.Security.PasswordCost)
		authMiddleware := middleware.NewAuthMiddleware(jwtManager)

		// 注册认证处理器
		authHandler := handlers.NewAuthHandler(db, jwtManager, hasher)
		authHandler.RegisterRoutes(apiV1)

		// 需要认证的路由组
		authenticated := apiV1.Group("")
		authenticated.Use(authMiddleware.RequireAuth())
		{
			// 注册用户管理处理器（需要管理员权限）
			userHandler := handlers.NewUserHandler(db, hasher)
			adminRoutes := authenticated.Group("")
			adminRoutes.Use(authMiddleware.RequireAdmin())
			userHandler.RegisterRoutes(adminRoutes)
		}
	}

	// 启动服务器
	addr := cfg.App.Host + ":" + strconv.Itoa(cfg.App.Port)
	logger.Info("Starting server",
		zap.String("address", addr),
		zap.String("environment", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
		zap.String("host", cfg.App.Host),
	)
	logger.Info("API endpoints",
		zap.String("api_base", "http://"+addr+"/api/v1"),
		zap.String("health_check", "http://"+addr+"/health"),
		zap.String("swagger_docs", "http://"+addr+"/swagger/index.html"),
	)

	// 优雅关机
	go func() {
		if err := router.Run(addr); err != nil {
			logger.Error("Failed to start server",
				zap.Error(err),
			)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 给服务器一些时间处理现有请求
	time.Sleep(3 * time.Second)
	logger.Info("Server exited")
}

// healthCheck 健康检查处理器
// @Summary 健康检查
// @Description 检查API服务状态
// @Tags 系统
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /health [get]
func healthCheck(c *gin.Context) {
	// 检查数据库连接
	if err := database.HealthCheck(); err != nil {
		c.JSON(503, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// rootHandler 根路径处理器
// @Summary API信息
// @Description 获取API基本信息
// @Tags 系统
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router / [get]
func rootHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Welcome to Admin System Go API",
		"version": "1.0.0",
		"docs":    "/api/v1/swagger", // 可以添加Swagger文档
		"health":  "/health",
	})
}

// createSuperUser 创建初始超级管理员
func createSuperUser() error {
	cfg := config.Get()

	db := database.DB()
	userRepo := repositories.NewUserRepository(db)

	// 检查超级管理员是否已存在
	existingUser, err := userRepo.GetByUsername(cfg.Superuser.Username)
	if err != nil {
		return err
	}

	if existingUser != nil {
		logger.Info("Super user already exists",
			zap.String("username", cfg.Superuser.Username),
		)
		return nil
	}

	// 创建超级管理员
	hasher := security.NewBCryptHasher(cfg.Security.PasswordCost)
	passwordHash, err := hasher.Hash(cfg.Superuser.Password)
	if err != nil {
		return err
	}

	superUser := &models.User{
		Username:     cfg.Superuser.Username,
		Email:        cfg.Superuser.Email,
		PasswordHash: passwordHash,
		FullName:     "System Administrator",
		IsActive:     true,
		Role:         models.RoleAdmin,
	}

	if err := userRepo.Create(superUser); err != nil {
		return err
	}

	logger.Info("Created super user",
		zap.String("username", cfg.Superuser.Username),
		zap.String("email", cfg.Superuser.Email),
	)
	return nil
}
