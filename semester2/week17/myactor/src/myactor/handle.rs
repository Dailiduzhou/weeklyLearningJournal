use tokio::sync::{mpsc, oneshot};

use super::{
    actor::{MyActor, run_my_actor},
    error::ActorError,
    message::ActorMsg,
};

#[derive(Clone)]
pub struct MyActorHandle {
    sender: mpsc::Sender<ActorMsg>,
}

impl Default for MyActorHandle {
    fn default() -> Self {
        Self::new()
    }
}

impl MyActorHandle {
    pub fn new() -> Self {
        let (sender, receiver) = mpsc::channel(10);
        let actor = MyActor::new(receiver);
        tokio::spawn(run_my_actor(actor));
        Self { sender }
    }

    pub async fn incr_counter(&self) -> Result<u32, ActorError> {
        let (reply_to, recv) = oneshot::channel();
        let message = ActorMsg::Incr { reply_to };

        self.sender
            .send(message)
            .await
            .map_err(|_| ActorError::MailboxClosed)?;

        recv.await.map_err(|_| ActorError::ActorStopped)
    }

    pub async fn incr_counter_by(&self, increment: u32) -> Result<u32, ActorError> {
        let (reply_to, recv) = oneshot::channel();
        let message = ActorMsg::IncrBy {
            increment,
            reply_to,
        };

        self.sender
            .send(message)
            .await
            .map_err(|_| ActorError::MailboxClosed)?;

        recv.await.map_err(|_| ActorError::ActorStopped)
    }
}

#[cfg(test)]
mod tests {
    use super::MyActorHandle;

    #[tokio::test]
    async fn counter_updates_across_messages() {
        let actor_handle = MyActorHandle::new();

        assert_eq!(actor_handle.incr_counter().await.unwrap(), 1);
        assert_eq!(actor_handle.incr_counter_by(32).await.unwrap(), 33);
    }
}
