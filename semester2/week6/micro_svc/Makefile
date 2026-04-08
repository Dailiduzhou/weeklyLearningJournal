.PHONY: help build up down logs test clean restart svc-b svc-a gateway etcd

# 默认目标
help:
	@echo "微服务本地测试工具"
	@echo ""
	@echo "使用方法:"
	@echo "  make up       - 启动所有服务"
	@echo "  make down     - 停止所有服务"
	@echo "  make restart  - 重启所有服务"
	@echo "  make logs     - 查看所有服务日志"
	@echo "  make test     - 测试 API 接口"
	@echo "  make clean    - 清理所有容器和数据卷"
	@echo ""
	@echo "单个服务操作:"
	@echo "  make etcd     - 只启动 etcd"
	@echo "  make svc-b    - 只启动服务 B"
	@echo "  make svc-a    - 只启动服务 A"
	@echo "  make gateway  - 只启动网关"

# 构建所有镜像
build:
	docker-compose build

# 启动所有服务
up:
	docker-compose up -d
	@echo "等待服务启动..."
	@sleep 5
	@echo "服务已启动，访问: http://localhost:8888/api/a/call"

# 停止所有服务
down:
	docker-compose down

# 停止并清理数据卷
clean:
	docker-compose down -v
	docker system prune -f

# 重启所有服务
restart: down up

# 查看日志
logs:
	docker-compose logs -f

# 查看单个服务日志
logs-etcd:
	docker-compose logs -f etcd

logs-svc-b:
	docker-compose logs -f svc-b

logs-svc-a:
	docker-compose logs -f svc-a

logs-gateway:
	docker-compose logs -f gateway

# 测试 API
test:
	@echo "测试 API..."
	@sleep 2
	@curl -s http://localhost:8888/api/a/call || echo "服务可能还未完全启动，请稍后再试"

# 测试多次
test-loop:
	@for i in 1 2 3 4 5; do \
		echo "第 $$i 次请求:"; \
		curl -s http://localhost:8888/api/a/call; \
		echo ""; \
		sleep 1; \
	done

# 进入容器调试
exec-etcd:
	docker-compose exec etcd etcdctl endpoint health

exec-svc-b:
	docker-compose exec svc-b sh

exec-svc-a:
	docker-compose exec svc-a sh

exec-gateway:
	docker-compose exec gateway sh

# 查看服务状态
ps:
	docker-compose ps

# 只启动 etcd
etcd:
	docker-compose up -d etcd
	@echo "etcd 已启动，地址: localhost:2379"

# 只启动服务 B
svc-b: etcd
	docker-compose up -d svc-b
	@echo "服务 B 已启动"

# 只启动服务 A
svc-a: svc-b
	docker-compose up -d svc-a
	@echo "服务 A 已启动"

# 只启动网关
gateway: svc-a
	docker-compose up -d gateway
	@echo "网关已启动，访问: http://localhost:8888"
