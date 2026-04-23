use std::collections::HashMap;
use tokio::sync::broadcast;

/// 房间信息
#[derive(Debug, Clone)]
pub struct RoomInfo {
    pub name: String,
    pub subscriber_count: usize,
}

/// 房间管理器
pub struct RoomManager {
    rooms: HashMap<String, broadcast::Sender<String>>,
    channel_capacity: usize,
}

impl RoomManager {
    /// 创建新的房间管理器
    pub fn new() -> Self {
        Self {
            rooms: HashMap::new(),
            channel_capacity: 100,
        }
    }

    /// 指定 channel 容量的构造函数（用于测试）
    #[cfg(test)]
    pub fn with_capacity(capacity: usize) -> Self {
        Self {
            rooms: HashMap::new(),
            channel_capacity: capacity,
        }
    }

    /// 获取或创建房间，返回发送端
    pub fn get_or_create(&mut self, name: &str) -> broadcast::Sender<String> {
        self.rooms
            .entry(name.to_string())
            .or_insert_with(|| {
                let (tx, _) = broadcast::channel(self.channel_capacity);
                tx
            })
            .clone()
    }

    /// 获取已存在房间的发送端，如果不存在返回 None
    pub fn get(&self, name: &str) -> Option<broadcast::Sender<String>> {
        self.rooms.get(name).cloned()
    }

    /// 检查房间是否存在
    pub fn exists(&self, name: &str) -> bool {
        self.rooms.contains_key(name)
    }

    /// 列出所有房间信息
    pub fn list_rooms(&self) -> Vec<RoomInfo> {
        self.rooms
            .iter()
            .map(|(name, tx)| RoomInfo {
                name: name.clone(),
                subscriber_count: tx.receiver_count(),
            })
            .collect()
    }

    /// 获取房间数量
    pub fn room_count(&self) -> usize {
        self.rooms.len()
    }

    /// 移除没有订阅者的空房间
    pub fn cleanup_empty_rooms(&mut self) {
        self.rooms.retain(|_, tx| tx.receiver_count() > 0);
    }

    /// 删除指定房间
    pub fn remove_room(&mut self, name: &str) -> bool {
        self.rooms.remove(name).is_some()
    }
}

impl Default for RoomManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_new_room_manager() {
        let manager = RoomManager::new();
        assert_eq!(manager.room_count(), 0);
        assert!(manager.list_rooms().is_empty());
    }

    #[test]
    fn test_create_new_room() {
        let mut manager = RoomManager::new();
        let tx = manager.get_or_create("test_room");
        
        assert_eq!(manager.room_count(), 1);
        assert!(manager.exists("test_room"));
        assert!(!manager.exists("other_room"));
    }

    #[test]
    fn test_get_existing_room() {
        let mut manager = RoomManager::new();
        let tx1 = manager.get_or_create("test_room");
        let tx2 = manager.get_or_create("test_room");
        
        // 验证是同一个 channel（通过发送消息测试）
        let mut rx = tx2.subscribe();
        tx1.send("hello".to_string()).unwrap();
        
        assert_eq!(rx.try_recv().unwrap(), "hello");
    }

    #[test]
    fn test_get_nonexistent_room() {
        let manager = RoomManager::new();
        assert!(manager.get("nonexistent").is_none());
    }

    #[test]
    fn test_list_rooms() {
        let mut manager = RoomManager::new();
        manager.get_or_create("room1");
        manager.get_or_create("room2");
        
        let rooms = manager.list_rooms();
        assert_eq!(rooms.len(), 2);
        
        let names: Vec<_> = rooms.iter().map(|r| r.name.clone()).collect();
        assert!(names.contains(&"room1".to_string()));
        assert!(names.contains(&"room2".to_string()));
    }

    #[test]
    fn test_list_rooms_with_subscribers() {
        let mut manager = RoomManager::new();
        let tx = manager.get_or_create("test_room");
        
        // 创建两个订阅者
        let _rx1 = tx.subscribe();
        let _rx2 = tx.subscribe();
        
        let rooms = manager.list_rooms();
        assert_eq!(rooms[0].subscriber_count, 2);
    }

    #[test]
    fn test_remove_room() {
        let mut manager = RoomManager::new();
        manager.get_or_create("test_room");
        
        assert!(manager.remove_room("test_room"));
        assert_eq!(manager.room_count(), 0);
        assert!(!manager.remove_room("test_room"));
    }

    #[test]
    fn test_cleanup_empty_rooms() {
        let mut manager = RoomManager::new();
        let tx = manager.get_or_create("empty_room");
        manager.get_or_create("active_room");
        
        // 为 active_room 添加订阅者
        let _rx = manager.get("active_room").unwrap().subscribe();
        
        // empty_room 没有订阅者，应该被清理
        manager.cleanup_empty_rooms();
        
        assert_eq!(manager.room_count(), 1);
        assert!(!manager.exists("empty_room"));
        assert!(manager.exists("active_room"));
    }

    #[test]
    fn test_multiple_rooms() {
        let mut manager = RoomManager::new();
        
        manager.get_or_create("room1");
        manager.get_or_create("room2");
        manager.get_or_create("room3");
        
        assert_eq!(manager.room_count(), 3);
        
        // 获取现有房间不应创建新的
        manager.get_or_create("room1");
        assert_eq!(manager.room_count(), 3);
    }

    #[tokio::test]
    async fn test_room_broadcast() {
        let mut manager = RoomManager::new();
        let tx = manager.get_or_create("broadcast_room");
        
        let mut rx1 = tx.subscribe();
        let mut rx2 = tx.subscribe();
        
        tx.send("test message".to_string()).unwrap();
        
        assert_eq!(rx1.recv().await.unwrap(), "test message");
        assert_eq!(rx2.recv().await.unwrap(), "test message");
    }
}
