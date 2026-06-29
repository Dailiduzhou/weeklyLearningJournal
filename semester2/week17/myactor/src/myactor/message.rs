use tokio::sync::oneshot;

pub enum ActorMsg {
    Incr {
        reply_to: oneshot::Sender<u32>,
    },
    IncrBy {
        increment: u32,
        reply_to: oneshot::Sender<u32>,
    },
}
