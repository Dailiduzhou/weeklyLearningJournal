use tokio::sync::mpsc;

use super::message::ActorMsg;

pub struct MyActor {
    receiver: mpsc::Receiver<ActorMsg>,
    counter: u32,
}

impl MyActor {
    pub fn new(receiver: mpsc::Receiver<ActorMsg>) -> Self {
        Self {
            receiver,
            counter: 0_u32,
        }
    }

    fn handle_message(&mut self, message: ActorMsg) {
        match message {
            ActorMsg::Incr { reply_to } => {
                self.counter += 1;
                let _ = reply_to.send(self.counter);
            }
            ActorMsg::IncrBy {
                increment,
                reply_to,
            } => {
                self.counter += increment;
                let _ = reply_to.send(self.counter);
            }
        }
    }
}

pub async fn run_my_actor(mut actor: MyActor) {
    while let Some(msg) = actor.receiver.recv().await {
        actor.handle_message(msg);
    }
}
