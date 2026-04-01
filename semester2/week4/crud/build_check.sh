#!/bin/bash

set -e

echo "===== 开始 Docker 构建诊断 ====="
echo

echo "1. 检查 Go 版本..."
go version
echo "✓ Go 版本正常"
echo

echo "2. 检查 go.mod..."
head -3 go.mod
echo "✓ go.mod 正常"
echo

echo "3. 检查依赖..."
go mod verify
echo "✓ 依赖验证通过"
echo

echo "4. 测试本地编译..."
cd cmd
go build -o user_test user.go
rm -f user_test
cd ..
echo "✓ 本地编译成功"
echo

echo "5. 清理 Docker 缓存..."
docker builder prune -f 2>/dev/null || echo "Docker 未运行，跳过"
echo

echo "6. 开始 Docker 构建..."
if docker build --progress=plain -t user-service:test .; then
	echo
	echo "✅ Docker 构建成功！"
	echo

	echo "7. 查看镜像信息..."
	docker images user-service:test
	echo

	echo "===== 构建完成 ====="
	echo
	echo "下一步操作："
	echo "  docker compose up -d      # 启动服务"
	echo "  make test-api             # 测试 API"
	echo
else
	echo
	echo "❌ Docker 构建失败"
	echo
	echo "故障排除步骤："
	echo "  1. 检查网络连接: ping goproxy.cn"
	echo "  2. 查看详细日志: DOCKER_BUILDKIT=0 docker build -t user-service:test ."
	echo "  3. 参考: TROUBLESHOOTING.md"
	exit 1
fi
