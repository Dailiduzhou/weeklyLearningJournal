//! 异步聊天服务器
//! 
//! 这是一个基于 Tokio 的异步聊天服务器，支持：
//! - 多房间聊天
//! - 实时消息广播
//! - 命令系统（/join, /rooms, /nick, /quit）

pub mod client;
pub mod command;
pub mod io;
pub mod room;
pub mod server;
