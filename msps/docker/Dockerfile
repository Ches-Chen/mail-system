# 使用官方Go镜像（根据你的Go版本选择，这里以1.20为例）
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum先处理依赖（分层构建优化缓存）
COPY ../go.mod ../go.sum ./

# 下载依赖
RUN go mod tidy

# 复制整个项目代码
COPY .. .

# 编译Go程序（替换成你的项目编译命令）
RUN CGO_ENABLED=0 GOOS=linux go build -o ./msps ./cmd/msps

# 切换到轻量级运行镜像（减少镜像体积）
FROM alpine:latest

# 设置运行时工作目录
WORKDIR /app

# 复制编译好的二进制文件
COPY --from=builder /app/msps ./

# 设置启动命令（替换成你的程序运行命令）
ENTRYPOINT ["./msps"]