.PHONY: help build run test clean docker-up docker-down docker-logs wire

help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## 编译应用
	@echo "🔨 编译应用..."
	@go build -o forum ./cmd

run: ## 运行应用
	@echo "🚀 启动应用..."
	@go run ./cmd/main.go

test: ## 运行测试
	@echo "🧪 运行测试..."
	@go test -v ./...

clean: ## 清理构建文件
	@echo "🧹 清理..."
	@rm -f forum
	@rm -f di/wire_gen.go

wire: ## 重新生成 Wire 代码
	@echo "🔧 生成 Wire 代码..."
	@cd di && wire

docker-build: ## 构建 Docker 镜像
	@echo "🐳 构建 Docker 镜像..."
	@docker build -t forum-api:latest .

docker-up: ## 启动 Docker Compose
	@echo "🚀 启动 Docker 服务..."
	@docker-compose up -d --build

docker-down: ## 停止 Docker Compose
	@echo "🛑 停止 Docker 服务..."
	@docker-compose down

docker-logs: ## 查看 Docker 日志
	@docker-compose logs -f

docker-restart: ## 重启 Docker 服务
	@echo "🔄 重启 Docker 服务..."
	@docker-compose restart

dev: ## 开发模式（本地运行）
	@echo "💻 开发模式..."
	@go run ./cmd/main.go

fmt: ## 格式化代码
	@echo "✨ 格式化代码..."
	@go fmt ./...

lint: ## 代码检查
	@echo "🔍 代码检查..."
	@golangci-lint run

deps: ## 安装依赖
	@echo "📦 安装依赖..."
	@go mod download
	@go mod tidy
