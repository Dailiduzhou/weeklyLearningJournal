# 快速开始指南

## 启动服务

直接运行启动脚本：
```bash
./start.sh
```

## 访问页面

在浏览器中打开：
**http://localhost:3000/index.html**

## 使用步骤

1. **生成 Token**
   - 在 "User ID" 输入框中输入用户ID（默认：12345）
   - 点击 "Generate Token" 按钮
   - 系统会自动生成一个 JWT token

2. **连接服务器**
   - 点击 "Connect" 按钮
   - 状态会从 "Disconnected" 变为 "Connected"

3. **查看比分**
   - 每隔 2 秒会自动更新比分信息
   - Team A 分数显示为绿色
   - Team B 分数显示为蓝色
   - 可以看到比赛时间和最新比分

## 停止服务

在运行 `start.sh` 的终端中按 `Ctrl+C` 停止所有服务。

## 注意事项

- 后端服务运行在 http://localhost:8080
- 前端服务运行在 http://localhost:3000
- JWT token 有效期为 2 小时
- 如果断开连接，点击 "Connect" 可重新连接，会自动接收错过的消息
