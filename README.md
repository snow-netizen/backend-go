# Admin System Go Backend

基于 Gin + GORM 的后台管理系统后端，提供用户认证、用户管理等功能，按照 REPLANNING.md 规范实现。

## 功能特性

- ✅ JWT 用户认证
- ✅ 基于角色的权限控制（RBAC）
- ✅ 用户管理（增删改查、分页、搜索）
- ✅ 密码加密存储（bcrypt）
- ✅ 验证码支持（基础实现）
- ✅ 登录失败锁定
- ✅ 数据库迁移（自动）
- ✅ 支持 SQLite / PostgreSQL
- ✅ 跨域支持（CORS）
- ✅ 健康检查接口
- ✅ 标准 RESTful API 设计

## 技术栈

- **框架**: Gin
- **数据库 ORM**: GORM
- **认证**: JWT
- **密码加密**: bcrypt
- **配置管理**: Viper
- **日志**: Zap
- **数据库**: SQLite（开发）、PostgreSQL（生产）
- **API 文档**: 计划集成 Swagger

## 快速开始

### 1. 安装依赖

```bash
# 进入项目目录
cd backend-go

# 下载依赖
go mod tidy

# 或者直接安装
go install ./...
```

### 2. 配置环境变量

```bash
# 复制环境变量示例文件
cp .env.example .env

# 编辑 .env 文件，配置数据库和密钥
vim .env
```

### 3. 启动开发服务器

```bash
# 方式一：直接运行
go run cmd/server/main.go

# 方式二：构建后运行
go build -o admin-system-go cmd/server/main.go
./admin-system-go
```

### 4. 访问 API

- 根路径: http://localhost:8080/
- 健康检查: http://localhost:8080/health
- API 基础路径: http://localhost:8080/api/v1
- **Swagger 文档**: http://localhost:8080/swagger/index.html

## API 接口

### 认证接口

- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/register` - 用户注册
- `GET /api/v1/auth/captcha` - 获取验证码
- `GET /api/v1/auth/profile` - 获取当前用户信息（需认证）
- `POST /api/v1/auth/logout` - 用户登出（需认证）
- `POST /api/v1/auth/change-password` - 修改密码（需认证）

### 用户管理接口（仅管理员）

- `GET /api/v1/users` - 获取用户列表（支持分页、搜索、筛选）
- `POST /api/v1/users` - 创建用户
- `GET /api/v1/users/:id` - 获取用户详情
- `PUT /api/v1/users/:id` - 更新用户
- `DELETE /api/v1/users/:id` - 删除用户
- `PUT /api/v1/users/:id/password` - 管理员修改用户密码
- `PUT /api/v1/users/:id/status` - 修改用户状态（启用/禁用）

## 初始管理员账号

系统启动时会自动创建初始管理员账号：

- **用户名**: admin
- **密码**: admin123
- **邮箱**: admin@example.com
- **角色**: admin

**注意**: 请在生产环境中修改默认密码！

## 项目结构

```
backend-go/
├── cmd/
│   └── server/
│       └── main.go          # 应用入口
├── internal/
│   ├── config/              # 配置管理
│   ├── database/            # 数据库连接和迁移
│   ├── models/              # 数据模型
│   ├── repositories/        # 数据访问层
│   ├── handlers/            # HTTP处理器
│   ├── middleware/          # 中间件（认证、日志等）
│   ├── security/            # 安全相关（密码、JWT）
│   └── services/            # 业务逻辑层（预留）
├── pkg/
│   ├── response/            # 响应格式化
│   └── validator/           # 数据验证（预留）
├── config/
│   └── config.yaml          # 配置文件
├── .env.example             # 环境变量示例
├── go.mod                   # Go模块定义
└── README.md               # 说明文档
```

## 配置说明

### 配置文件 (config/config.yaml)

```yaml
app:
  name: "admin-system-go"
  version: "1.0.0"
  env: "development"  # development, production
  port: 8080
  host: "0.0.0.0"

jwt:
  secret: "your-secret-key-change-in-production"
  issuer: "admin-system-go"
  access_token_expiry: 3600    # 1小时
  refresh_token_expiry: 604800 # 7天

database:
  driver: "sqlite"  # sqlite, postgres
  dsn: "admin_system.db"

security:
  password_cost: 12  # bcrypt cost
  captcha:
    length: 6
    expiry: 300      # 5分钟
  login:
    max_failures: 5
    lock_duration: 900  # 15分钟

cors:
  allowed_origins:
    - "http://localhost:3000"
    - "http://localhost:5173"

superuser:
  username: "admin"
  email: "admin@example.com"
  password: "admin123"
```

### 环境变量优先级

环境变量会覆盖配置文件中的值：
- `DATABASE_URL` - 数据库连接字符串
- `DB_DRIVER` - 数据库驱动
- `JWT_SECRET` - JWT密钥
- `APP_ENV` - 应用环境
- `PORT` - 端口号
- `CORS_ORIGINS` - CORS允许的源（逗号分隔）

## 数据库

### 支持的数据驱动

1. **SQLite** (开发环境默认)
   ```yaml
   database:
     driver: "sqlite"
     dsn: "admin_system.db"
   ```

2. **PostgreSQL** (生产环境推荐)
   ```yaml
   database:
     driver: "postgres"
     dsn: "postgresql://user:password@localhost/admin_system?sslmode=disable"
   ```

### 数据库表结构

系统自动创建以下表：
- `users` - 用户表
- `captcha_codes` - 验证码表
- `login_logs` - 登录日志表
- `operation_logs` - 操作日志表

## 部署

### 开发环境

```bash
# 使用SQLite，快速启动
go run cmd/server/main.go
```

### Swagger API 文档

系统集成了Swagger API文档，启动后可以通过以下方式访问：

1. **在线文档**: http://localhost:8080/swagger/index.html
2. **JSON格式**: http://localhost:8080/swagger/doc.json
3. **YAML格式**: http://localhost:8080/swagger/doc.yaml

**重新生成文档**（在修改API注释后）：

```bash
# 重新生成Swagger文档
swag init -g cmd/server/main.go -o docs
```

### 生产环境

```bash
# 1. 构建应用
go build -o admin-system-go -ldflags="-s -w" cmd/server/main.go

# 2. 配置生产环境变量
export APP_ENV=production
export JWT_SECRET=your-production-secret
export DATABASE_URL=postgresql://...
export DB_DRIVER=postgres

# 3. 启动应用
./admin-system-go
```

### 使用 Docker

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN go build -o admin-system-go cmd/server/main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 复制可执行文件
COPY --from=builder /app/admin-system-go .
COPY --from=builder /app/config ./config

# 设置环境变量
ENV APP_ENV=production

# 暴露端口
EXPOSE 8080

# 启动应用
CMD ["./admin-system-go"]
```

## 测试

```bash
# 运行测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 运行特定包的测试
go test ./internal/handlers
```

## 与 Python 版本的对比

| 特性 | Python (FastAPI) 版本 | Go (Gin) 版本 |
|------|----------------------|---------------|
| 框架 | FastAPI + SQLAlchemy | Gin + GORM |
| 性能 | 良好 | 优秀（高并发） |
| 开发速度 | 快速 | 中等 |
| 内存使用 | 较高 | 较低 |
| 部署大小 | 较大 | 较小 |
| 生态 | Python 丰富生态 | Go 标准库强大 |

## 下一步计划

1. ✅ 基础框架搭建
2. ✅ 用户认证功能
3. ✅ 用户管理功能
4. ✅ 验证码图片生成
5. ⬜ 操作日志记录
6. ✅ Swagger API 文档
7. ⬜ 单元测试覆盖
8. ⬜ Docker 部署配置

## 许可证

MIT