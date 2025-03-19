# StellarFrp API 服务器

这是一个使用Go语言开发的高性能API系统，支持异步操作、MySQL数据库集成和性能优化。

## 特性

- 基于Gin框架的RESTful API
- 异步处理机制
- MySQL数据库集成与连接池优化
- Redis缓存支持
- 结构化日志记录
- 中间件支持（认证、日志、错误处理等）
- 模块化架构设计

## 项目结构

```
.
├── cmd/                # 应用程序入口
│   └── server/         # API服务器入口
├── config/             # 配置文件和配置加载
├── internal/           # 内部包
│   ├── api/            # API处理器
│   ├── middleware/     # HTTP中间件
│   ├── model/          # 数据模型
│   ├── repository/     # 数据访问层
│   └── service/        # 业务逻辑层
├── pkg/                # 可重用的库
│   ├── async/          # 异步处理工具
│   ├── database/       # 数据库连接和工具
│   ├── logger/         # 日志工具
│   └── validator/      # 数据验证工具
└── scripts/            # 脚本和工具
```

## 快速开始

### 前置条件

- Go 1.21+
- MySQL 5.7+
- Redis 6.0+

### 安装

```bash
# 克隆仓库
git clone https://github.com/65658dsf/StellarServerGo.git
cd server

# 安装依赖
go mod tidy
```

### 配置

创建一个`.env`文件在项目根目录：

```
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=stellarfrp

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

API_PORT=8080
LOG_LEVEL=info
```

### 运行

```bash
go run cmd/server/main.go
```

## API文档

启动服务器后，访问 `http://localhost:8080/swagger/index.html` 查看API文档。

## 许可证

MIT