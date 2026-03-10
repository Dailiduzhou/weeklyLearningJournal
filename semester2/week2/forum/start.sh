#!/bin/bash

# Forum API Docker 快速启动脚本

set -e

echo "🚀 启动 Forum API..."

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker"
    exit 1
fi

# 检查 docker-compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose 未安装"
    exit 1
fi

# 启动服务
echo "📦 构建并启动容器..."
docker-compose up -d --build

# 等待服务就绪
echo "⏳ 等待服务启动..."
sleep 5

# 检查服务状态
if docker-compose ps | grep -q "Up"; then
    echo "✅ 服务启动成功！"
    echo ""
    echo "📍 服务地址:"
    echo "   - API: http://localhost:8080"
    echo "   - MongoDB: mongodb://admin:admin123@localhost:27017"
    echo ""
    echo "📝 查看日志: docker-compose logs -f"
    echo "🛑 停止服务: docker-compose down"
else
    echo "❌ 服务启动失败，请检查日志"
    docker-compose logs
    exit 1
fi
