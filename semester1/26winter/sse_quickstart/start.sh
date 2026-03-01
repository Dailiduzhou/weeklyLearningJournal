#!/bin/bash

echo "🚀 Starting SSE Server and Frontend Demo..."
echo ""

# Check if .env exists, if not, create from .env.example
if [ ! -f .env ]; then
	echo "📝 Creating .env from .env.example..."
	cp .env.example .env
fi

# Kill any existing processes on ports 8080 and 3000
echo "🧹 Cleaning up existing processes..."
lsof -ti:8080 | xargs -r kill -9 2>/dev/null
lsof -ti:3000 | xargs -r kill -9 2>/dev/null
sleep 1
echo "✓ Cleanup complete"
echo ""

# Start backend server in background
echo "🔧 Starting backend server on port 8080..."
go run main.go &
BACKEND_PID=$!

echo "Backend PID: $BACKEND_PID"
echo ""

# Wait for backend to start
echo "⏳ Waiting for backend to start..."
sleep 2

# Start frontend server
echo "🎨 Starting frontend server on port 3000..."
python3 -m http.server 3000 --directory frontend &
FRONTEND_PID=$!

echo "Frontend PID: $FRONTEND_PID"
echo ""

echo "✅ Servers started successfully!"
echo ""
echo "📊 Backend:  http://localhost:8080"
echo "🌐 Frontend: http://localhost:3000/index.html"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📝 使用说明 / Instructions:"
echo ""
echo "1️⃣  在浏览器中打开: http://localhost:3000/index.html"
echo "2️⃣  点击 'Generate Token' 生成 JWT token"
echo "3️⃣  点击 'Connect' 连接到 SSE 服务器"
echo "4️⃣  每隔 2 秒会自动更新比分信息"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Wait a bit more for servers to be fully ready
sleep 2

# Auto-open browser based on OS
if command -v xdg-open >/dev/null; then
	xdg-open http://localhost:3000/index.html 2>/dev/null
elif command -v open >/dev/null; then
	open http://localhost:3000/index.html 2>/dev/null
elif command -v start >/dev/null; then
	start http://localhost:3000/index.html 2>/dev/null
fi
echo "Press Ctrl+C to stop both servers"

# Handle cleanup on exit
cleanup() {
	echo ""
	echo "🛑 Stopping servers..."
	kill $BACKEND_PID 2>/dev/null
	kill $FRONTEND_PID 2>/dev/null
	exit 0
}

trap cleanup SIGINT SIGTERM

# Keep script running
wait
