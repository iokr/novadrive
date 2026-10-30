# NovaDrive 构建与开发命令
APP    := novadrive
BIN    := bin/$(APP)
CONFIG ?= config/config.yaml
WEB    := frontend
DIST   := web/dist

.PHONY: help build build-all web web-install run dev tidy fmt vet test clean

help:
	@echo "make web-install  # 安装前端依赖（npm ci）"
	@echo "make web          # 构建前端，产物输出到 $(DIST)"
	@echo "make build        # 编译单二进制到 $(BIN)（内嵌现有 $(DIST)）"
	@echo "make build-all    # 前端 + 后端一键构建"
	@echo "make run          # 直接运行（默认配置 $(CONFIG)）"
	@echo "make dev          # 前端开发服务器（/api 代理到 :8080）"
	@echo "make tidy         # 整理依赖"
	@echo "make fmt          # 格式化代码"
	@echo "make vet          # 静态检查"
	@echo "make test         # 运行测试"
	@echo "make clean        # 清理构建产物"

web-install:
	cd $(WEB) && npm ci

web:
	cd $(WEB) && npm run build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/$(APP)

build-all: web build

run:
	go run ./cmd/$(APP) -c $(CONFIG)

dev:
	cd $(WEB) && npm run dev

tidy:
	go mod tidy

fmt:
	gofmt -l -w .

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf bin
