# 聊天室测试指南

## 启动服务器
```bash
go run server.go
```

## 测试步骤

### 1. 打开浏览器开发者工具
- 按F12打开开发者工具
- 切换到"Console"标签

### 2. 访问聊天页面
- 打开浏览器访问：http://localhost:8080
- 输入用户名并点击"Join Chat"

### 3. 观察控制台输出
你应该看到以下日志：
- `Received raw data:` - 收到服务器消息
- `Parsed message:` - 解析后的消息对象
- `Sending message:` - 发送消息时的日志

### 4. 测试发送消息
- 在消息输入框中输入文字
- 点击"Send"按钮或按回车
- 观察控制台和聊天窗口

### 5. 测试多用户
- 打开第二个浏览器标签页
- 使用不同的用户名加入
- 在一个标签页发送消息，检查另一个标签页是否收到

## 调试页面

使用 `debug.html` 进行更详细的调试：
```bash
# 在浏览器中打开
open http://localhost:8080/debug.html
```

这个页面会显示：
- WebSocket连接状态
- 发送和接收的所有消息
- 详细的错误信息

## 常见问题

### 问题1：无法连接WebSocket
- 检查服务器是否运行：`ps aux | grep server.go`
- 检查端口是否被占用：`lsof -i :8080`

### 问题2：消息发送后没有显示
- 打开浏览器控制台查看错误
- 检查WebSocket状态：`ws.readyState` (1 = OPEN)

### 问题3：看不到其他用户
- 确保使用不同的用户名登录
- 检查控制台是否有"Received raw data"日志

## 服务器日志

服务器会显示以下日志：
- `New connection from xxx` - 新用户连接
- `Received message from xxx` - 收到消息
- `Sent message to xxx` - 发送消息
- `Read error from xxx` - 读取错误
- `Write error to xxx` - 写入错误
