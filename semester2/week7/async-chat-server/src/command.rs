/// 聊天命令类型
#[derive(Debug, Clone, PartialEq)]
pub enum Command {
    /// 发送普通消息到当前房间
    Message(String),
    /// 加入指定房间 /join <room>
    Join(String),
    /// 列出所有房间 /rooms
    Rooms,
    /// 修改昵称 /nick <name>
    Nick(String),
    /// 退出 /quit
    Quit,
    /// 未知命令
    Unknown(String),
}

impl Command {
    /// 解析输入字符串为命令
    pub fn parse(input: &str) -> Self {
        let trimmed = input.trim();
        
        if trimmed.is_empty() {
            return Command::Message(String::new());
        }
        
        if !trimmed.starts_with('/') {
            return Command::Message(trimmed.to_string());
        }
        
        let parts: Vec<&str> = trimmed.splitn(2, ' ').collect();
        let cmd = parts[0];
        let arg = parts.get(1).map(|s| s.trim());
        
        match cmd {
            "/join" => {
                match arg {
                    Some(room) if !room.is_empty() => Command::Join(room.to_string()),
                    _ => Command::Unknown("Usage: /join <room_name>".to_string()),
                }
            }
            "/rooms" => Command::Rooms,
            "/nick" => {
                match arg {
                    Some(name) if !name.is_empty() => Command::Nick(name.to_string()),
                    _ => Command::Unknown("Usage: /nick <nickname>".to_string()),
                }
            }
            "/quit" => Command::Quit,
            _ => Command::Unknown(format!("Unknown command: {}", cmd)),
        }
    }
    
    /// 获取命令的简短描述（用于帮助信息）
    pub fn description(&self) -> &'static str {
        match self {
            Command::Message(_) => "发送消息",
            Command::Join(_) => "加入房间",
            Command::Rooms => "列出所有房间",
            Command::Nick(_) => "修改昵称",
            Command::Quit => "退出服务器",
            Command::Unknown(_) => "未知命令",
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_message() {
        assert_eq!(
            Command::parse("hello world"),
            Command::Message("hello world".to_string())
        );
    }

    #[test]
    fn test_parse_join() {
        assert_eq!(
            Command::parse("/join room1"),
            Command::Join("room1".to_string())
        );
    }

    #[test]
    fn test_parse_join_without_arg() {
        assert_eq!(
            Command::parse("/join"),
            Command::Unknown("Usage: /join <room_name>".to_string())
        );
    }

    #[test]
    fn test_parse_join_empty_arg() {
        assert_eq!(
            Command::parse("/join   "),
            Command::Unknown("Usage: /join <room_name>".to_string())
        );
    }

    #[test]
    fn test_parse_rooms() {
        assert_eq!(Command::parse("/rooms"), Command::Rooms);
    }

    #[test]
    fn test_parse_nick() {
        assert_eq!(
            Command::parse("/nick alice"),
            Command::Nick("alice".to_string())
        );
    }

    #[test]
    fn test_parse_nick_without_arg() {
        assert_eq!(
            Command::parse("/nick"),
            Command::Unknown("Usage: /nick <nickname>".to_string())
        );
    }

    #[test]
    fn test_parse_quit() {
        assert_eq!(Command::parse("/quit"), Command::Quit);
    }

    #[test]
    fn test_parse_unknown_command() {
        assert_eq!(
            Command::parse("/unknown"),
            Command::Unknown("Unknown command: /unknown".to_string())
        );
    }

    #[test]
    fn test_parse_empty() {
        assert_eq!(Command::parse(""), Command::Message(String::new()));
    }

    #[test]
    fn test_parse_whitespace_only() {
        assert_eq!(Command::parse("   "), Command::Message(String::new()));
    }

    #[test]
    fn test_parse_message_with_leading_trailing_space() {
        assert_eq!(
            Command::parse("  hello world  "),
            Command::Message("hello world".to_string())
        );
    }

    #[test]
    fn test_command_descriptions() {
        assert_eq!(Command::Message(String::new()).description(), "发送消息");
        assert_eq!(Command::Join(String::new()).description(), "加入房间");
        assert_eq!(Command::Rooms.description(), "列出所有房间");
        assert_eq!(Command::Nick(String::new()).description(), "修改昵称");
        assert_eq!(Command::Quit.description(), "退出服务器");
    }
}
