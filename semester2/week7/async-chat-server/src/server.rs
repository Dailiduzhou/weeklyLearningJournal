use crate::client::ClientSession;
use crate::io::{AsyncReadWrite, TcpIo};
use crate::room::RoomManager;
use anyhow::Result;
use std::sync::Arc;
use tokio::net::TcpListener;
use tokio::sync::{Mutex, watch};

/// 服务器配置
#[derive(Debug, Clone)]
pub struct ServerConfig {
    pub welcome_message: String,
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            welcome_message: "欢迎来到聊天服务器！".to_string(),
        }
    }
}

/// 聊天服务器
pub struct ChatServer {
    listener: TcpListener,
    room_manager: Arc<Mutex<RoomManager>>,
    config: ServerConfig,
}

impl ChatServer {
    /// 创建新的服务器实例
    pub fn new(
        listener: TcpListener,
        room_manager: Arc<Mutex<RoomManager>>,
    ) -> Self {
        Self {
            listener,
            room_manager,
            config: ServerConfig::default(),
        }
    }

    /// 设置自定义配置
    pub fn with_config(mut self, config: ServerConfig) -> Self {
        self.config = config;
        self
    }

    /// 运行服务器
    pub async fn run(self, mut shutdown_rx: watch::Receiver<bool>) -> Result<()> {
        println!("Server listening on {}", self.listener.local_addr()?);

        loop {
            tokio::select! {
                result = self.listener.accept() => {
                    let (socket, addr) = result?;
                    println!("[{}] 已连接", addr);

                    let room_manager_clone = Arc::clone(&self.room_manager);
                    let shutdown_rx_clone = shutdown_rx.clone();
                    let config = self.config.clone();

                    tokio::spawn(async move {
                        if let Err(e) = Self::handle_connection(
                            socket,
                            addr,
                            room_manager_clone,
                            shutdown_rx_clone,
                            config,
                        ).await {
                            eprintln!("[{}] 客户端异常: {}", addr, e);
                        }
                    });
                }
                _ = shutdown_rx.changed() => {
                    println!("[Server] 停止接受新连接。");
                    break;
                }
            }
        }
        Ok(())
    }

    /// 处理单个连接
    async fn handle_connection(
        socket: tokio::net::TcpStream,
        addr: std::net::SocketAddr,
        room_manager: Arc<Mutex<RoomManager>>,
        shutdown_rx: watch::Receiver<bool>,
        config: ServerConfig,
    ) -> Result<()> {
        let (read_half, write_half) = socket.into_split();
        let io = TcpIo::new(read_half, write_half);
        
        let mut session = ClientSession::new(io, addr, room_manager).await;
        
        // 发送欢迎消息
        session.io.writeln(&config.welcome_message).await?;
        session.send_welcome().await?;
        
        // 运行会话
        session.run(shutdown_rx).await
    }
}

/// 兼容旧 API 的函数
pub async fn run_server(
    listener: TcpListener,
    room_manager: Arc<Mutex<RoomManager>>,
    shutdown_rx: watch::Receiver<bool>,
) -> Result<()> {
    let server = ChatServer::new(listener, room_manager);
    server.run(shutdown_rx).await
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_server_config_default() {
        let config = ServerConfig::default();
        assert!(!config.welcome_message.is_empty());
    }

    #[test]
    fn test_server_config_custom() {
        let config = ServerConfig {
            welcome_message: "Custom welcome".to_string(),
        };
        assert_eq!(config.welcome_message, "Custom welcome");
    }

    #[tokio::test]
    async fn test_room_manager_creation() {
        let manager = RoomManager::new();
        assert_eq!(manager.room_count(), 0);
    }
}
