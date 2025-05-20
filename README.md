# Mail System 邮件系统

一个现代化的邮件系统解决方案，包含邮件服务端、邮件发送代理和Web客户端。

## 项目架构

项目由三个主要模块组成：

- **MSPS (Mail Service Provider System)**: 邮件服务提供系统，基于Go语言开发的后端服务
  - 使用Gin框架构建RESTful API
  - MySQL数据库存储
  - 支持Docker部署

- **Agent**: 邮件发送代理程序
  - 基于Go语言开发
  - 支持多种邮件服务商
  - 内置健康检查机制

- **Web Client**: 基于Vue 3的现代化Web客户端
  - 使用Vite作为构建工具
  - 响应式设计
  - 用户友好的界面

## 技术栈

### 后端
- Go 1.23.0
- Gin Web Framework
- MySQL
- Docker

### 前端
- Vue 3
- Vite

### 代理程序
- Go 1.23.0
- go-mail
- go-resty

## 快速开始

### 1. 启动MSPS服务

```bash
# 进入msps目录
cd msps

# 使用Docker启动服务
docker-compose up -d
```

### 2. 配置并启动Agent

```bash
# 进入agent目录
cd agent

# 运行agent
go run main.go
```

### 3. 启动Web客户端

```bash
# 进入web客户端目录
cd client/web

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

## 项目结构

```
.
├── agent/                 # 邮件发送代理
├── client/               # 客户端应用
│   └── web/             # Web客户端
└── msps/                 # 邮件服务提供系统
    ├── cmd/             # 主程序入口
    ├── docker/          # Docker相关配置
    ├── docs/            # API文档
    ├── internal/        # 内部包
    └── test/            # 测试文件
```

## 许可证

[MIT License](LICENSE)