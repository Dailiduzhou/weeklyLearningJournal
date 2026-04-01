#!/bin/bash

echo "===== 验证 PostgreSQL 驱动修复 ====="
echo

echo "1. 检查 go.mod 中的驱动依赖..."
if grep -q "github.com/lib/pq" go.mod; then
	echo "✓ go.mod 包含 github.com/lib/pq"
	grep "github.com/lib/pq" go.mod
else
	echo "✗ go.mod 缺少驱动依赖"
	echo "  运行: go get github.com/lib/pq"
fi
echo

echo "2. 检查驱动导入..."
if grep -q "_ \"github.com/lib/pq\"" cmd/user.go; then
	echo "✓ cmd/user.go 包含驱动导入"
else
	echo "✗ cmd/user.go 缺少驱动导入"
	echo "  添加: import _ \"github.com/lib/pq\""
fi
echo

echo "3. 测试本地编译..."
if cd cmd && go build -o user_verify user.go 2>&1; then
	echo "✓ 本地编译成功"
	rm -f user_verify
	cd ..
else
	echo "✗ 本地编译失败"
	cd ..
	exit 1
fi
echo

echo "4. 检查 Dockerfile..."
if grep -q "COPY go.mod go.sum" Dockerfile; then
	echo "✓ Dockerfile 包含依赖复制步骤"
else
	echo "✗ Dockerfile 可能缺少依赖复制"
fi
echo

echo "===== 验证完成 ====="
echo
echo "下一步："
echo "  docker compose build --no-cache"
echo "  docker compose up -d"
