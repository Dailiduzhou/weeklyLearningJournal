use crate::command::Command;
use crate::io::AsyncReadWrite;
use crate::room::RoomManager;
use anyhow::Result;
use std::sync::Arc;
use tokio::sync::{Mutex, broadcast, watch};
use tokio::time::{timeout, Duration};

/// 客户端会话状态
#[derive(Debug, Clone)]
pub struct ClientState {
    pub nickname: String,
    pub current_room: String,
}

impl ClientState {
    pub fn new(nickname: String) -> Self {
        Self {
            nickname,
            current_room: "#general".to_string(),
        }
    }
}

/// 客户端会话处理器
pub struct ClientSession<R: AsyncReadWrite> {
    pub io: R,
    state: ClientState,
    room_manager: Arc<Mutex<RoomManager>>,
    pub room_tx: broadcast::Sender<String>,
    room_rx: broadcast::Receiver<String>,
    addr: std::net::SocketAddr,
}

impl<R: AsyncReadWrite> ClientSession<R> {
    /// 创建新的客户端会话
    pub async fn new(
        io: R,
        addr: std::net::SocketAddr,
        room_manager: Arc<Mutex<RoomManager>>,
    ) -> Self {
        let nickname = addr.to_string();
        let state = ClientState::new(nickname.clone());
        
        // 加入默认房间
        let room_tx = {
            let mut manager = room_manager.lock().await;
            manager.get_or_create(&state.current_room)
        };
        let room_rx = room_tx.subscribe();
        
        Self {
            io,
            state,
            room_manager,
            room_tx,
            room_rx,
            addr,
        }
    }

    /// 设置自定义昵称
    pub fn set_nickname(&mut self, nickname: String) {
        self.state.nickname = nickname;
    }

    /// 获取当前状态
    pub fn state(&self) -> &ClientState {
        &self.state
    }

    /// 发送欢迎消息
    pub async fn send_welcome(&mut self) -> Result<()> {
        self.io.writeln(&format!("欢迎来到 {}！", self.state.current_room)).await?;
        self.broadcast_to_room(&format!("* {} 加入了 {}", self.state.nickname, self.state.current_room))
            .await;
        Ok(())
    }

    /// 运行客户端主循环
    pub async fn run(mut self, mut shutdown_rx: watch::Receiver<bool>) -> Result<()> {
        loop {
            tokio::select! {
                // 处理用户输入
                result = timeout(Duration::from_secs(300), self.io.read_line()) => {
                    match result {
                        Ok(Ok(Some(line))) => {
                            let cmd = Command::parse(&line);
                            if self.handle_command(cmd).await? {
                                break; // 退出命令
                            }
                        }
                        Ok(Ok(None)) => break, // EOF
                        Ok(Err(e)) => {
                            eprintln!("[{}] 读取错误: {}", self.addr, e);
                            break;
                        }
                        Err(_) => {
                            self.io.writeln("由于长时间未活动，您已断开连接。").await?;
                            break;
                        }
                    }
                }
                // 接收广播消息
                result = self.room_rx.recv() => {
                    match result {
                        Ok(msg) => {
                            let _ = self.io.write(&msg).await;
                        }
                        Err(broadcast::error::RecvError::Lagged(n)) => {
                            let _ = self.io
                                .writeln(&format!("* 警告: 错过了 {} 条消息。", n))
                                .await;
                        }
                        Err(broadcast::error::RecvError::Closed) => break,
                    }
                }
                // 停机信号
                _ = shutdown_rx.changed() => {
                    let _ = self.io.writeln("服务器正在关闭...").await;
                    break;
                }
            }
        }

        // 发送退出消息
        self.broadcast_to_room(&format!("* {} 退出了服务器", self.state.nickname))
            .await;
        println!("[{}] 已断开连接", self.addr);
        Ok(())
    }

    /// 处理命令，返回 true 表示应该退出
    async fn handle_command(&mut self, cmd: Command) -> Result<bool> {
        match cmd {
            Command::Message(msg) => {
                if !msg.is_empty() {
                    self.broadcast_to_room(&format!("{}: {}", self.state.nickname, msg))
                        .await;
                }
                Ok(false)
            }
            Command::Join(room) => {
                self.handle_join(room).await?;
                Ok(false)
            }
            Command::Rooms => {
                self.handle_rooms().await?;
                Ok(false)
            }
            Command::Nick(name) => {
                self.handle_nick(name).await?;
                Ok(false)
            }
            Command::Quit => Ok(true),
            Command::Unknown(msg) => {
                self.io.writeln(&format!("错误: {}", msg)).await?;
                Ok(false)
            }
        }
    }

    /// 处理加入房间命令
    async fn handle_join(&mut self, new_room: String) -> Result<()> {
        // 离开当前房间
        self.broadcast_to_room(&format!("* {} 离开了 {}", self.state.nickname, self.state.current_room))
            .await;

        // 加入新房间
        self.state.current_room = new_room;
        self.room_tx = {
            let mut manager = self.room_manager.lock().await;
            manager.get_or_create(&self.state.current_room)
        };
        self.room_rx = self.room_tx.subscribe();

        // 通知新房间
        self.broadcast_to_room(&format!("* {} 加入了 {}", self.state.nickname, self.state.current_room))
            .await;

        Ok(())
    }

    /// 处理列出房间命令
    async fn handle_rooms(&mut self) -> Result<()> {
        let manager = self.room_manager.lock().await;
        let rooms = manager.list_rooms();
        
        let mut response = String::from("活跃的房间:\n");
        for room in &rooms {
            response.push_str(&format!("- {} ({} 人在线)\n", room.name, room.subscriber_count));
        }
        
        if rooms.is_empty() {
            response.push_str("(暂无活跃房间)\n");
        }
        
        self.io.write(&response).await?;
        Ok(())
    }

    /// 处理修改昵称命令
    async fn handle_nick(&mut self, new_name: String) -> Result<()> {
        let old_name = self.state.nickname.clone();
        self.state.nickname = new_name.clone();
        
        self.broadcast_to_room(&format!("* {} 改名为 {}", old_name, new_name))
            .await;
        self.io.writeln(&format!("昵称已修改为: {}", new_name)).await?;
        
        Ok(())
    }

    /// 向当前房间广播消息
    async fn broadcast_to_room(&self, msg: &str) {
        let _ = self.room_tx.send(format!("{}\n", msg));
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::io::MockIo;

    async fn create_test_session(inputs: Vec<&str>) -> ClientSession<MockIo> {
        let mock_io = MockIo::new(inputs);
        let room_manager = Arc::new(Mutex::new(RoomManager::new()));
        let addr = "127.0.0.1:12345".parse().unwrap();
        
        ClientSession::new(mock_io, addr, room_manager).await
    }

    #[tokio::test]
    async fn test_client_state_new() {
        let state = ClientState::new("test_user".to_string());
        assert_eq!(state.nickname, "test_user");
        assert_eq!(state.current_room, "#general");
    }

    #[tokio::test]
    async fn test_session_creation() {
        let session = create_test_session(vec![]).await;
        assert_eq!(session.state.nickname, "127.0.0.1:12345");
        assert_eq!(session.state.current_room, "#general");
    }

    #[tokio::test]
    async fn test_set_nickname() {
        let mut session = create_test_session(vec![]).await;
        session.set_nickname("alice".to_string());
        assert_eq!(session.state.nickname, "alice");
    }

    #[tokio::test]
    async fn test_handle_message_command() {
        let mut session = create_test_session(vec![]).await;
        session.set_nickname("tester".to_string());
        
        // 先订阅频道
        let mut rx = session.room_tx.subscribe();
        
        let cmd = Command::Message("hello world".to_string());
        let should_quit = session.handle_command(cmd).await.unwrap();
        
        assert!(!should_quit);
        // 验证消息被广播（通过检查是否有接收者收到）
        assert!(rx.try_recv().unwrap().contains("tester: hello world"));
    }

    #[tokio::test]
    async fn test_handle_join_command() {
        let mut session = create_test_session(vec![]).await;
        session.set_nickname("tester".to_string());
        
        let cmd = Command::Join("new_room".to_string());
        let should_quit = session.handle_command(cmd).await.unwrap();
        
        assert!(!should_quit);
        assert_eq!(session.state.current_room, "new_room");
    }

    #[tokio::test]
    async fn test_handle_rooms_command() {
        let mut session = create_test_session(vec![]).await;
        
        // 先创建一些房间
        {
            let mut manager = session.room_manager.lock().await;
            manager.get_or_create("room1");
            manager.get_or_create("room2");
        }
        
        let cmd = Command::Rooms;
        let should_quit = session.handle_command(cmd).await.unwrap();
        
        assert!(!should_quit);
        assert!(session.io.contains_output("room1"));
        assert!(session.io.contains_output("room2"));
    }

    #[tokio::test]
    async fn test_handle_nick_command() {
        let mut session = create_test_session(vec![]).await;
        
        let cmd = Command::Nick("alice".to_string());
        let should_quit = session.handle_command(cmd).await.unwrap();
        
        assert!(!should_quit);
        assert_eq!(session.state.nickname, "alice");
        assert!(session.io.contains_output("alice"));
    }

    #[tokio::test]
    async fn test_handle_quit_command() {
        let mut session = create_test_session(vec![]).await;
        
        let cmd = Command::Quit;
        let should_quit = session.handle_command(cmd).await.unwrap();
        
        assert!(should_quit);
    }

    #[tokio::test]
    async fn test_handle_unknown_command() {
        let mut session = create_test_session(vec![]).await;
        
        let cmd = Command::Unknown("test error".to_string());
        let should_quit = session.handle_command(cmd).await.unwrap();
        
        assert!(!should_quit);
        assert!(session.io.contains_output("test error"));
    }

}
