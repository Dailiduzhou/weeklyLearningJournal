use anyhow::Result;
use async_trait::async_trait;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::net::tcp::{OwnedReadHalf, OwnedWriteHalf};

/// IO 读写抽象 trait
#[async_trait]
pub trait AsyncReadWrite: Send {
    /// 读取一行输入，返回 None 表示 EOF
    async fn read_line(&mut self) -> Result<Option<String>>;
    /// 写入消息
    async fn write(&mut self, msg: &str) -> Result<()>;
    /// 写入并换行
    async fn writeln(&mut self, msg: &str) -> Result<()> {
        self.write(&format!("{}\n", msg)).await
    }
}

/// TCP 流的 IO 包装器
pub struct TcpIo {
    reader: BufReader<OwnedReadHalf>,
    writer: OwnedWriteHalf,
}

impl TcpIo {
    pub fn new(read_half: OwnedReadHalf, write_half: OwnedWriteHalf) -> Self {
        Self {
            reader: BufReader::new(read_half),
            writer: write_half,
        }
    }
}

#[async_trait]
impl AsyncReadWrite for TcpIo {
    async fn read_line(&mut self) -> Result<Option<String>> {
        let mut line = String::new();
        match self.reader.read_line(&mut line).await? {
            0 => Ok(None),
            _ => Ok(Some(line)),
        }
    }

    async fn write(&mut self, msg: &str) -> Result<()> {
        self.writer.write_all(msg.as_bytes()).await?;
        self.writer.flush().await?;
        Ok(())
    }
}

/// Mock IO 实现，用于测试
#[derive(Debug)]
pub struct MockIo {
    inputs: Vec<String>,
    input_index: usize,
    outputs: Vec<String>,
}

impl MockIo {
    pub fn new(inputs: Vec<&str>) -> Self {
        Self {
            inputs: inputs.into_iter().map(|s| s.to_string()).collect(),
            input_index: 0,
            outputs: Vec::new(),
        }
    }

    pub fn outputs(&self) -> &[String] {
        &self.outputs
    }

    pub fn contains_output(&self, pattern: &str) -> bool {
        self.outputs.iter().any(|o| o.contains(pattern))
    }

    pub fn clear_outputs(&mut self) {
        self.outputs.clear();
    }
}

#[async_trait]
impl AsyncReadWrite for MockIo {
    async fn read_line(&mut self) -> Result<Option<String>> {
        if self.input_index < self.inputs.len() {
            let input = self.inputs[self.input_index].clone();
            self.input_index += 1;
            Ok(Some(input + "\n"))
        } else {
            Ok(None) // EOF
        }
    }

    async fn write(&mut self, msg: &str) -> Result<()> {
        self.outputs.push(msg.to_string());
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_mock_io_creation() {
        let mock = MockIo::new(vec!["hello", "world"]);
        assert_eq!(mock.outputs().len(), 0);
    }

    #[tokio::test]
    async fn test_mock_io_read_line() {
        let mut mock = MockIo::new(vec!["hello", "world"]);
        
        assert_eq!(mock.read_line().await.unwrap(), Some("hello\n".to_string()));
        assert_eq!(mock.read_line().await.unwrap(), Some("world\n".to_string()));
        assert_eq!(mock.read_line().await.unwrap(), None);
    }

    #[tokio::test]
    async fn test_mock_io_write() {
        let mut mock = MockIo::new(vec![]);
        
        mock.write("test message").await.unwrap();
        assert_eq!(mock.outputs().len(), 1);
        assert!(mock.contains_output("test message"));
    }

    #[tokio::test]
    async fn test_mock_io_contains_output() {
        let mut mock = MockIo::new(vec![]);
        
        mock.write("hello world").await.unwrap();
        assert!(mock.contains_output("world"));
        assert!(!mock.contains_output("foo"));
    }

    #[tokio::test]
    async fn test_mock_io_clear_outputs() {
        let mut mock = MockIo::new(vec![]);
        
        mock.write("test").await.unwrap();
        mock.clear_outputs();
        assert_eq!(mock.outputs().len(), 0);
    }
}
