//! 集成测试
//! 
//! 这些测试启动一个真实的服务器并通过 TCP 连接进行测试

use async_chat_server::room::RoomManager;
use async_chat_server::server::ChatServer;
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpStream;
use tokio::sync::{Mutex, watch};
use tokio::time::{timeout, Duration};

/// 测试服务器辅助结构
pub struct TestServer {
    pub addr: std::net::SocketAddr,
    shutdown_tx: watch::Sender<bool>,
}

impl TestServer {
    /// 启动测试服务器
    pub async fn start() -> Self {
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();

        let room_manager = Arc::new(Mutex::new(RoomManager::new()));
        let (shutdown_tx, shutdown_rx) = watch::channel(false);

        let server = ChatServer::new(listener, room_manager);

        tokio::spawn(async move {
            let _ = server.run(shutdown_rx).await;
        });

        // 等待服务器启动
        tokio::time::sleep(Duration::from_millis(100)).await;

        Self {
            addr,
            shutdown_tx,
        }
    }

    /// 连接到服务器
    pub async fn connect(&self) -> TcpStream {
        timeout(Duration::from_secs(5), TcpStream::connect(self.addr))
            .await
            .expect("连接超时")
            .expect("连接失败")
    }

    /// 关闭服务器
    pub fn shutdown(self) {
        let _ = self.shutdown_tx.send(true);
    }
}

/// 测试客户端辅助结构
pub struct TestClient {
    stream: TcpStream,
    buffer: Vec<u8>,
}

impl TestClient {
    pub fn new(stream: TcpStream) -> Self {
        Self {
            stream,
            buffer: vec![0u8; 1024],
        }
    }

    /// 发送消息
    pub async fn send(&mut self, msg: &str) {
        self.stream
            .write_all(format!("{}\n", msg).as_bytes())
            .await
            .expect("发送失败");
    }

    /// 读取响应（返回读取到的字符串）
    pub async fn recv(&mut self) -> String {
        let n = timeout(Duration::from_secs(2), self.stream.read(&mut self.buffer))
            .await
            .expect("读取超时")
            .expect("读取失败");
        String::from_utf8_lossy(&self.buffer[..n]).to_string()
    }

    /// 读取多行直到超时（用于收集多条消息）
    pub async fn recv_multiple(&mut self, timeout_ms: u64) -> Vec<String> {
        let mut messages = Vec::new();
        let deadline = tokio::time::Instant::now() + Duration::from_millis(timeout_ms);

        while tokio::time::Instant::now() < deadline {
            match timeout(Duration::from_millis(100), self.stream.read(&mut self.buffer)).await {
                Ok(Ok(n)) if n > 0 => {
                    messages.push(String::from_utf8_lossy(&self.buffer[..n]).to_string());
                }
                _ => break,
            }
        }

        messages
    }

    /// 检查是否收到包含指定文本的消息
    pub async fn expect(&mut self, pattern: &str) {
        let msg = self.recv().await;
        assert!(
            msg.contains(pattern),
            "期望包含 '{}' 的消息，但收到: {}",
            pattern,
            msg
        );
    }
}

#[tokio::test]
async fn test_server_start_and_client_connect() {
    let server = TestServer::start().await;
    let mut client = TestClient::new(server.connect().await);

    // 应该收到欢迎消息和加入房间消息
    let welcome = client.recv().await;
    assert!(welcome.contains("欢迎"), "应该收到欢迎消息");

    server.shutdown();
}

#[tokio::test]
async fn test_client_join_room() {
    let server = TestServer::start().await;
    let mut client = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client.recv_multiple(200).await;

    // 发送加入房间命令
    client.send("/join test_room").await;

    // 应该收到离开旧房间和加入新房间的消息
    let messages = client.recv_multiple(200).await;
    let combined: String = messages.concat();
    assert!(combined.contains("离开") || combined.contains("test_room"), "应该收到房间变更消息");

    server.shutdown();
}

#[tokio::test]
async fn test_message_broadcast() {
    let server = TestServer::start().await;

    // 连接两个客户端
    let mut client1 = TestClient::new(server.connect().await);
    let mut client2 = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client1.recv_multiple(200).await;
    let _ = client2.recv_multiple(200).await;

    // client1 发送消息
    client1.send("hello from client1").await;

    // client2 应该收到消息
    let msg = client2.recv().await;
    assert!(
        msg.contains("hello from client1"),
        "client2 应该收到 client1 的消息"
    );

    server.shutdown();
}

#[tokio::test]
async fn test_change_nickname() {
    let server = TestServer::start().await;
    let mut client = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client.recv_multiple(200).await;

    // 修改昵称
    client.send("/nick alice").await;

    // 应该收到确认消息
    let msg = client.recv().await;
    assert!(msg.contains("alice"), "应该收到昵称修改确认");

    server.shutdown();
}

#[tokio::test]
async fn test_list_rooms() {
    let server = TestServer::start().await;
    let mut client = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client.recv_multiple(200).await;

    // 请求房间列表
    client.send("/rooms").await;

    // 应该收到房间列表
    let msg = client.recv().await;
    assert!(msg.contains("活跃的房间") || msg.contains("#general"), "应该收到房间列表");

    server.shutdown();
}

#[tokio::test]
async fn test_multiple_clients_in_different_rooms() {
    let server = TestServer::start().await;

    // 连接两个客户端
    let mut client1 = TestClient::new(server.connect().await);
    let mut client2 = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client1.recv_multiple(200).await;
    let _ = client2.recv_multiple(200).await;

    // client1 加入不同的房间
    client1.send("/join room_a").await;
    let _ = client1.recv_multiple(200).await; // 等待加入完成

    // client1 发送消息
    client1.send("message in room_a").await;

    // client2 在默认房间，不应该收到 room_a 的消息
    // 给一点时间让消息传播
    tokio::time::sleep(Duration::from_millis(100)).await;

    // client2 发送一个消息来触发读取超时
    client2.send("/rooms").await;
    let response = client2.recv().await;
    assert!(!response.contains("room_a"), "client2 不应该看到 room_a 的消息");

    server.shutdown();
}

#[tokio::test]
async fn test_client_quit() {
    let server = TestServer::start().await;

    // 连接两个客户端
    let mut client1 = TestClient::new(server.connect().await);
    let mut client2 = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client1.recv_multiple(200).await;
    let _ = client2.recv_multiple(200).await;

    // client1 退出
    client1.send("/quit").await;

    // 给一点时间让退出消息传播
    tokio::time::sleep(Duration::from_millis(100)).await;

    // client2 发送命令检查是否还在运行
    client2.send("/rooms").await;
    let response = client2.recv().await;
    assert!(!response.is_empty(), "服务器应该仍在运行");

    server.shutdown();
}

#[tokio::test]
async fn test_unknown_command() {
    let server = TestServer::start().await;
    let mut client = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client.recv_multiple(200).await;

    // 发送未知命令
    client.send("/unknowncommand").await;

    // 应该收到错误消息
    let msg = client.recv().await;
    assert!(
        msg.contains("Unknown") || msg.contains("未知") || msg.contains("错误"),
        "应该收到错误消息: {}",
        msg
    );

    server.shutdown();
}

#[tokio::test]
async fn test_empty_message() {
    let server = TestServer::start().await;
    let mut client = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client.recv_multiple(200).await;

    // 发送空消息（只有换行）
    client.send("").await;

    // 发送一条正常消息确认连接还在
    client.send("/rooms").await;
    let response = client.recv().await;
    assert!(!response.is_empty(), "服务器应该仍在运行");

    server.shutdown();
}

#[tokio::test]
async fn test_join_room_without_name() {
    let server = TestServer::start().await;
    let mut client = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client.recv_multiple(200).await;

    // 发送不带房间名的加入命令
    client.send("/join").await;

    // 应该收到使用帮助
    let msg = client.recv().await;
    assert!(
        msg.contains("Usage") || msg.contains("用法") || msg.contains("错误"),
        "应该收到用法提示: {}",
        msg
    );

    server.shutdown();
}

#[tokio::test]
async fn test_nick_without_name() {
    let server = TestServer::start().await;
    let mut client = TestClient::new(server.connect().await);

    // 跳过初始消息
    let _ = client.recv_multiple(200).await;

    // 发送不带昵称的 nick 命令
    client.send("/nick").await;

    // 应该收到使用帮助
    let msg = client.recv().await;
    assert!(
        msg.contains("Usage") || msg.contains("用法") || msg.contains("错误"),
        "应该收到用法提示: {}",
        msg
    );

    server.shutdown();
}
