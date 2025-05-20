APP=msps
VERSION=latest
WIRE_DIR=internal/app/injector

.PHONY: help
## help: 获取帮助文档
help:
	@echo "Usage: "
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: build
build: export GOARCH=amd64
build: export CGO_ENABLED=1
build: export CC=gcc
build: export CXX=g++
build: init
	@go build -ldflags "-linkmode external -extldflags -static -w -s" -o target/ ./cmd/${APP}

.PHONY: init
## init: 项目初始化
init: export GOPROXY=https://goproxy.cn
init: export GO111MODULE=on
init:
	@go mod tidy && go mod vendor

.PHONY: docker
## docker: 构建docker镜像
docker:
	@docker build -t ${APP}:${VERSION} .

.PHONY: wire
## wire: 自动依赖注入
wire: export GOPROXY=https://goproxy.cn
wire: export GO111MODULE=on
wire:
	@go install github.com/google/wire/cmd/wire@latest
	@cd ${WIRE_DIR} && wire

.PHONY: swag
## swag: 构建API文档
swag:
	@swag init -g ./cmd/${APP}/main.go
