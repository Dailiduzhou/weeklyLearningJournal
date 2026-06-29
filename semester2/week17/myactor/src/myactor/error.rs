use std::fmt::{self, Display, Formatter};

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ActorError {
    MailboxClosed,
    ActorStopped,
}

impl Display for ActorError {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        match self {
            ActorError::MailboxClosed => f.write_str("actor mailbox is closed"),
            ActorError::ActorStopped => f.write_str("actor stopped before replying"),
        }
    }
}

impl std::error::Error for ActorError {}
