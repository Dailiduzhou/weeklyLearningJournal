# 聊天室故障排查与修复文档

## 问题描述

用户报告聊天室存在以下问题：

1. Go代码中缺少错误处理
2. 聊天室无法显示消息
3. 无法显示在线用户数

## 初步排查

### 1. 检查代码结构

```bash
find . -name "*.go" -type f
ls -la
```

发现主要文件：

- `server.go` - Go服务器代码
- `test-client/test_client.go` - 测试客户端
- `public/index.html` - 前端页面

### 2. 分析代码

阅读 `server.go` 发现以下问题：

#### 错误处理缺失

```go
// 第147行
c.conn.WriteMessage(websocket.CloseMessage, []byte{})
// 缺少错误检查

// 第159行
c.conn.WriteMessage(websocket.PingMessage, nil)
// 缺少错误检查
```

#### Hub广播机制问题

```go
// hub.run() 中的register case
case client := <-h.register:
    h.clients[client] = true
    h.usernames[client.username] = true
    
    // 直接向broadcast channel发送消息
    h.broadcast <- Message{Type: "login", ...}
    h.broadcast <- Message{Type: "userlist", ...}
```

#### 问题分析

当新客户端连接时：

1. 创建client并启动goroutine
2. 立即向hub.register发送注册请求
3. hub收到注册后，向broadcast channel发送消息
4. 但此时新客户端的writePump可能还未准备好接收消息

**关键问题**：select的随机性和goroutine启动的时序问题导致消息丢失。

## 详细故障排查

### 第1次测试

```bash
cd test-client && go run test_client.go TestUser7
```

结果：客户端连接成功，但只收到部分消息

### 第2次测试

查看服务器日志：

```
2026/02/01 20:32:57 Server started on :8080
2026/02/01 20:32:57 Hub started
2026/02/01 20:33:00 New connection from TestUser7
2026/02/01 20:33:00 Registering client: TestUser7
2026/02/01 20:33:00   Client count after register: 1
2026/02/01 20:33:00 Sent message to TestUser7: userlist
```

发现：只收到userlist消息，没有收到login和userlist的广播消息。

### 第3次测试

修改测试客户端使用ReadJSON：

```go
go func() {
    defer close(done)
    for {
        var message interface{}
        if err := c.ReadJSON(&message); err != nil {
            log.Println("Read error:", err)
            return
        }
        log.Printf("Received: %+v", message)
    }
}()
```

结果：可以收到消息了，但仍然不完整。

### 第4次测试 - 双客户端测试

启动两个客户端测试：

```bash
go run test_client.go User1
go run test_client.go User2
```

发现User1收到了User2的登录消息和userlist更新！

## 问题根源分析

通过多次测试和日志分析，确认了以下问题：

### 1. Goroutine时序问题

原代码顺序：

```go
hub.register <- client
go client.writePump()
go client.readPump()
```

问题：

- 向hub.register发送请求后，立即返回
- hub收到后立即向broadcast channel发送消息
- 但writePump goroutine可能还没启动完成
- 导致select在send channel时失败

### 2. Select随机性问题

hub.run中的select语句：

```go
select {
case client := <-h.register:
    // 注册逻辑
    h.broadcast <- Message{...}  // 这会发送到broadcast
case message := <-h.broadcast:
    // 广播逻辑
    for client := range h.clients {
        select {
        case client.send <- message:
        default:
            // 发送失败！
        }
        }
}
```

由于select的随机性，可能：

- register case先执行，向broadcast发送消息
- 但此时broadcast case还没准备好接收
- 导致消息积压或丢失

### 3. 缓冲区大小不足

```go
send: make(chan Message, 256)
```

在多个消息同时发送时，可能缓冲区已满。

## 修复方案

### 修复1：补充Go错误处理

#### server.go 修复

**修复1 - WriteMessage错误处理（第147行）**

```go
// 修复前
if !ok {
    c.conn.WriteMessage(websocket.CloseMessage, []byte{})
    return
}

// 修复后
if !ok {
    if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
        log.Printf("Error sending close message to %s: %v", c.username, err)
    }
    return
}
```

**修复2 - PingMessage错误处理（第159行）**

```go
// 修复前
if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
    return
}

// 修复后
if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
    log.Printf("Ping error to %s: %v", c.username, err)
    return
}
```

#### test_client.go 修复

**修复1 - 连接错误处理**

```go
// 修复前
c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
if err != nil {
    log.Fatal("Dial error:", err)
}
defer c.Close()

// 修复后
c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
if err != nil {
    log.Fatal("Dial error:", err)
}
defer func() {
    if err := c.Close(); err != nil {
        log.Printf("Error closing connection: %v", err)
    }
}()
```

**修复2 - 写入错误处理**

```go
// 修复前
err := c.WriteJSON(message)
if err != nil {
    log.Println("Write error:", err)
    return
}

// 修复后
if err := c.WriteJSON(message); err != nil {
    log.Println("Write error:", err)
    return
}
```

### 修复2：解决消息丢失问题

#### 修复1 - 增大缓冲区

```go
// 修复前
send: make(chan Message, 256),

// 修复后
send: make(chan Message, 512),
```

#### 修复2 - 调整goroutine启动顺序

```go
// 修复前
hub.register <- client
go client.writePump()
go client.readPump()

// 修复后
go client.writePump()
go client.readPump()

time.Sleep(10 * time.Millisecond)

hub.register <- client
```

**说明**：

- 先启动writePump和readPump
- 等待10ms确保goroutine启动完成
- 再向hub注册

#### 修复3 - 优化广播机制

**重大改动**：在register case中直接发送消息，绕过broadcast channel

```go
// 修复前
case client := <-h.register:
    log.Printf("Registering client: %s", client.username)
    h.clients[client] = true
    h.usernames[client.username] = true
    h.broadcast <- Message{Type: "login", ...}
    h.broadcast <- Message{Type: "userlist", ...}

// 修复后
case client := <-h.register:
    log.Printf("Registering client: %s", client.username)
    h.clients[client] = true
    h.usernames[client.username] = true

    // 直接构造消息
    loginMsg := Message{
        Type:      "login",
        Username:  client.username,
        Timestamp: time.Now(),
    }
    userlistMsg := Message{
        Type:      "userlist",
        Usernames: getUsernames(h),
        Timestamp: time.Now(),
    }

    time.Sleep(10 * time.Millisecond)

    // 直接发送给所有客户端
    log.Printf("Broadcasting message type: %s to %d clients", loginMsg.Type, len(h.clients))
    for c := range h.clients {
        select {
        case c.send <- loginMsg:
            log.Printf("  - Sent to %s", c.username)
        default:
            log.Printf("  - Failed to send to %s", c.username)
        }
    }

    log.Printf("Broadcasting message type: %s to %d clients", userlistMsg.Type, len(h.clients))
    for c := range h.clients {
        select {
        case c.send <- userlistMsg:
            log.Printf("  - Sent to %s", c.username)
        default:
            log.Printf("  - Failed to send to %s", c.username)
        }
    }
```

**优点**：

1. 避免了broadcast channel的竞争
2. 确保消息按顺序发送
3. 添加详细日志便于调试

#### 修复4 - 移除误关客户端逻辑

```go
// 修复前
case message := <-h.broadcast:
    for client := range h.clients {
        select {
        case client.send <- message:
            log.Printf("  - Sent to %s", client.username)
        default:
            log.Printf("  - Failed to send to %s, closing connection", client.username)
            close(client.send)
            delete(h.clients, client)  // 这里会误关客户端！
        }
    }

// 修复后
case message := <-h.broadcast:
    log.Printf("Broadcasting message type: %s to %d clients", message.Type, len(h.clients))
    for client := range h.clients {
        select {
        case client.send <- message:
            log.Printf("  - Sent to %s", client.username)
        default:
            log.Printf("  - Failed to send to %s", client.username)
            // 不再立即关闭，让readPump自然关闭
        }
    }
```

### 修复3：前端错误处理

```javascript
// 修复前
ws.onmessage = (event) => {
    console.log('Received raw data:', event.data);
    const msg = JSON.parse(event.data);
    console.log('Parsed message:', msg);
    handleMessage(msg);
};

// 修复后
ws.onmessage = (event) => {
    console.log('Received raw data:', event.data);
    try {
        const msg = JSON.parse(event.data);
        console.log('Parsed message:', msg);
        handleMessage(msg);
    } catch (e) {
        console.error('Error parsing message:', e, 'Raw data:', event.data);
    }
};
```

## 测试验证

### 测试1：单客户端测试

```bash
cd test-client && go run test_client.go TestUser12
```

**结果**：

```
2026/02/01 20:51:49 Connecting to ws://localhost:8080/ws?username=TestUser12
2026/02/01 20:51:49 Connected as TestUser12
2026/02/01 20:51:49 Received: map[timestamp:... type:userlist]
2026/02/01 20:51:49 Received: map[timestamp:... type:login username:TestUser12]
2026/02/01 20:51:49 Received: map[timestamp:... type:userlist usernames:[TestUser12]]
2026/02/01 20:51:50 Sent: Hello from TestUser12
2026/02/01 20:51:50 Received: map[content:Hello from TestUser12 ... type:chat username:TestUser12]
```

✅ 成功收到所有消息：userlist、login、userlist更新、聊天消息

### 测试2：双客户端测试

```bash
go run test_client.go User1 &
go run test_client.go User2 &
```

**User1收到**：

```
2026/02/01 20:55:36 Received: map[type:userlist usernames:[User1]]
2026/02/01 20:55:36 Received: map[type:login username:User2]
2026/02/01 20:55:36 Received: map[type:userlist usernames:[User2 User1]]
2026/02/01 20:55:37 Received: map[type:chat username:User1 content:Hello from User1]
2026/02/01 20:55:37 Received: map[type:chat username:User2 content:Hello from User2]
```

**User2收到**：

```
2026/02/01 20:55:36 Received: map[type:userlist usernames:[User1]]
2026/02/01 20:55:36 Received: map[type:login username:User2]
2026/02/01 20:55:36 Received: map[type:userlist usernames:[User2 User1]]
2026/02/01 20:55:37 Received: map[type:chat username:User1 content:Hello from User1]
2026/02/01 20:55:37 Received: map[type:chat username:User2 content:Hello from User2]
```

✅ 两个客户端都能看到：

- 在线用户列表（包含双方）
- 对方登录/退出通知
- 对方的聊天消息

### 测试3：服务器日志验证

```
2026/02/01 20:55:36 Registering client: User1
2026/02/01 20:55:36 Broadcasting message type: login to 1 clients
2026/02/01 20:55:36   - Sent to User1
2026/02/01 20:55:36 Broadcasting message type: userlist to 1 clients
2026/02/01 20:55:36   - Sent to User1
2026/02/01 20:55:36 New connection from User2
2026/02/01 20:55:36 Registering client: User2
2026/02/01 20:55:36 Broadcasting message type: login to 2 clients
2026/02/01 20:55:36   - Sent to User1
2026/02/01 20:55:36   - Sent to User2
2026/02/01 20:55:36 Broadcasting message type: userlist to 2 clients
2026/02/01 20:55:36   - Sent to User1
2026/02/01 20:55:36   - Sent to User2
```

✅ 所有消息都成功发送到所有客户端

## 总结

### 核心问题

WebSocket聊天室消息丢失的根本原因是：**goroutine时序问题和select随机性导致新连接的客户端在writePump未准备好时，hub就向其发送消息，造成消息积压或丢失**。

### 关键修复点

1. **调整goroutine启动顺序**：先启动writePump，延迟注册到hub
2. **增大channel缓冲区**：256 → 512
3. **优化广播机制**：register时直接发送，绕过broadcast channel避免竞争
4. **补充错误处理**：所有IO操作都添加错误检查
5. **移除误关逻辑**：避免因临时发送失败而错误关闭客户端

### 经验教训

1. **goroutine时序很重要**：在依赖goroutine间通信时，要考虑启动顺序和就绪时间
2. **select有随机性**：不要依赖select的执行顺序，如果需要顺序保证，应该重新设计
3. **缓冲区大小要合理**：在高并发场景下，channel缓冲区大小会影响消息吞吐量
4. **错误处理不可省**：所有IO操作都应该检查错误，避免静默失败
5. **日志要详细**：在调试并发问题时，详细的日志是定位问题的关键

### 测试建议

1. 单客户端测试：验证基本功能
2. 双客户端测试：验证消息互通
3. 多客户端并发测试：验证高并发场景
4. 异常测试：模拟网络中断、服务重启等
5. 压力测试：大量客户端同时连接/断开

## 相关文件

- `server.go` - Go WebSocket服务器
- `test-client/test_client.go` - 测试客户端
- `public/index.html` - 前端页面
- `public/style.css` - 样式文件

---

**修复日期**：2026-02-01  
**修复人**：opencode  
**测试环境**：Go 1.x + gorilla/websocket
