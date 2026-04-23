mod client;
mod command;
mod io;
mod room;
mod server;

use anyhow::Result;
use room::RoomManager;
use server::ChatServer;
use std::sync::Arc;
use tokio::net::TcpListener;
use tokio::sync::{Mutex, watch};
use tokio::time::Duration;

#[tokio::main]
async fn main() -> Result<()> {
    let (shutdown_tx, shutdown_rx) = watch::channel(false);

    // 优雅停机信号拦截
    tokio::spawn(async move {
        tokio::signal::ctrl_c().await.unwrap();
        println!("\n[System] 收到退出信号，准备优雅停机...");
        let _ = shutdown_tx.send(true);
    });

    let listener = TcpListener::bind("127.0.0.1:8080").await?;
    
    // 初始化全局状态
    let room_manager = Arc::new(Mutex::new(RoomManager::new()));

    // 创建并启动服务器
    let server = ChatServer::new(listener, room_manager);
    server.run(shutdown_rx).await?;

    tokio::time::sleep(Duration::from_millis(100)).await;
    println!("Server exit.");
    Ok(())
}

#[cfg(test)]
mod integration_tests {
    use super::*;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};
    use tokio::net::TcpStream;
    use tokio::time::timeout;

    /// 启动测试服务器
    async fn start_test_server() -> (std::net::SocketAddr, watch::Sender<bool>) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        
        let room_manager = Arc::new(Mutex::new(RoomManager::new()));
        let (shutdown_tx, shutdown_rx) = watch::channel(false);
        
        let server = ChatServer::new(listener, room_manager);
        
        tokio::spawn(async move {
            let _ = server.run(shutdown_rx).await;
        });
        
        // 等待服务器启动
        tokio::time::sleep(Duration::from_millis(100)).await;
        
        (addr, shutdown_tx)
    }

    /// 连接到测试服务器
    async fn connect_client(addr: std::net::SocketAddr) -> TcpStream {
        timeout(Duration::from_secs(5), TcpStream::connect(addr))
            .await
            .expect("连接超时")
            .expect("连接失败")
    }

    /// 读取响应（累积数据直到超时）
    async fn read_response(stream: &mut TcpStream, timeout_ms: u64) -> String {
        let mut buf = vec![0u8; 1024];
        let mut result = String::new();
        
        let deadline = tokio::time::Instant::now() + Duration::from_millis(timeout_ms);
        
        while tokio::time::Instant::now() < deadline {
            match timeout(Duration::from_millis(50), stream.read(&mut buf)).await {
                Ok(Ok(0)) => break, // EOF
                Ok(Ok(n)) => {
                    result.push_str(&String::from_utf8_lossy(&buf[..n]));
                    // 如果收到了换行符，就认为是一行完整的消息
                    if result.contains('\n') {
                        break;
                    }
                }
                Ok(Err(_)) => break,
                Err(_) => break, // 超时，返回已收到的数据
            }
        }
        
        result
    }

    #[tokio::test]
    async fn test_server_start_and_connect() {
        let (addr, _shutdown_tx) = start_test_server().await;
        
        let mut client = connect_client(addr).await;
        let response = read_response(&mut client, 500).await;
        
        assert!(response.contains("欢迎"));
    }

    #[tokio::test]
    async fn test_client_join_room() {
        let (addr, _shutdown_tx) = start_test_server().await;
        
        let mut client = connect_client(addr).await;
        let _welcome = read_response(&mut client, 200).await; // 跳过欢迎消息
        let _join_msg = read_response(&mut client, 200).await; // 跳过加入消息
        
        // 发送加入房间命令
        client.write_all(b"/join test_room\n").await.unwrap();
        
        // 读取加入新房间消息（离开原房间的消息不会收到，因为已经切换到新房间）
        let join_msg = read_response(&mut client, 500).await;
        assert!(join_msg.contains("test_room"), "应该收到加入消息: {}", join_msg);
        
        // 发送消息验证在新房间
        client.write_all(b"hello from test_room\n").await.unwrap();
        let echo = read_response(&mut client, 500).await;
        assert!(echo.contains("hello from test_room"), "应该收到自己的消息: {}", echo);
    }

    #[tokio::test]
    async fn test_client_send_message() {
        let (addr, _shutdown_tx) = start_test_server().await;
        
        // 连接两个客户端
        let mut client1 = connect_client(addr).await;
        let mut client2 = connect_client(addr).await;
        
        // 跳过欢迎和加入消息
        for _ in 0..2 {
            let _ = read_response(&mut client1, 200).await;
            let _ = read_response(&mut client2, 200).await;
        }
        
        // client1 发送消息
        client1.write_all(b"hello everyone\n").await.unwrap();
        
        // client2 应该收到消息
        let msg = read_response(&mut client2, 500).await;
        assert!(msg.contains("hello everyone"), "client2 应该收到消息: {}", msg);
    }

    #[tokio::test]
    async fn test_client_change_nickname() {
        let (addr, _shutdown_tx) = start_test_server().await;
        
        let mut client = connect_client(addr).await;
        
        // 跳过欢迎和加入消息
        for _ in 0..2 {
            let _ = read_response(&mut client, 200).await;
        }
        
        // 修改昵称
        client.write_all(b"/nick alice\n").await.unwrap();
        
        // 读取改名消息
        let nick_msg = read_response(&mut client, 500).await;
        assert!(nick_msg.contains("alice"), "应该收到改名消息: {}", nick_msg);
    }

    #[tokio::test]
    async fn test_list_rooms() {
        let (addr, _shutdown_tx) = start_test_server().await;
        
        let mut client = connect_client(addr).await;
        
        // 跳过欢迎和加入消息
        for _ in 0..2 {
            let _ = read_response(&mut client, 200).await;
        }
        
        // 请求房间列表
        client.write_all(b"/rooms\n").await.unwrap();
        
        let response = read_response(&mut client, 500).await;
        assert!(response.contains("活跃的房间"), "应该收到房间列表: {}", response);
    }

    #[tokio::test]
    async fn test_server_shutdown() {
        let (addr, shutdown_tx) = start_test_server().await;
        
        let mut client = connect_client(addr).await;
        let _ = read_response(&mut client, 500).await; // 欢迎消息
        
        // 发送停机信号
        let _ = shutdown_tx.send(true);
        
        // 等待服务器处理
        tokio::time::sleep(Duration::from_millis(200)).await;
        
        // 新连接应该失败
        let result = timeout(Duration::from_secs(1), TcpStream::connect(addr)).await;
        assert!(result.is_err() || result.unwrap().is_err());
    }
}
