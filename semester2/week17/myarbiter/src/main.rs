use actix::prelude::*;

struct SumActor {}

impl Actor for SumActor {
    type Context = Context<Self>;
}

#[derive(Message)]
#[rtype(result = "usize")]
struct Value(usize, usize);

impl Handler<Value> for SumActor {
    type Result = usize;

    fn handle(&mut self, msg: Value, _ctx: &mut Context<Self>) -> Self::Result {
        msg.0 + msg.1
    }
}

struct DisplayActor {}

impl Actor for DisplayActor {
    type Context = Context<Self>;
}

#[derive(Message)]
#[rtype(result = "()")]
struct Display(usize);

impl Handler<Display> for DisplayActor {
    type Result = ();

    fn handle(&mut self, msg: Display, _ctx: &mut Context<Self>) -> Self::Result {
        println!("Got {:?}", msg.0);
    }
}

fn main() {
    let system = System::new();

    // Define an execution flow using futures
    let execution = async {
        // `Actor::start` spawns the `Actor` on the *current* `Arbiter`, which
        // in this case is the System arbiter
        let sum_addr = SumActor {}.start();
        let dis_addr = DisplayActor {}.start();

        // Start by sending a `Value(6, 7)` to our `SumActor`.
        // `Addr::send` responds with a `Request`, which implements `Future`.
        // When awaited, it will resolve to a `Result<usize, MailboxError>`.
        let sum_result = sum_addr.send(Value(6, 7)).await;

        match sum_result {
            Ok(res) => {
                // `res` is now the `usize` returned from `SumActor` as a response to `Value(6, 7)`
                // Once the future is complete, send the successful response (`usize`)
                // to the `DisplayActor` wrapped in a `Display`
                dis_addr.send(Display(res)).await;
            }
            Err(e) => {
                eprintln!("Encountered mailbox error: {:?}", e);
            }
        };
    };

    // Spawn the future onto the current Arbiter/event loop
    Arbiter::current().spawn(execution);

    // We only want to do one computation in this example, so we
    // shut down the `System` which will stop any Arbiters within
    // it (including the System Arbiter), which will in turn stop
    // any Actor Contexts running within those Arbiters, finally
    // shutting down all Actors.
    System::current().stop();

    system.run();
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::mpsc;
    use std::time::Duration;

    struct ProbeActor {
        tx: mpsc::Sender<usize>,
    }

    impl Actor for ProbeActor {
        type Context = Context<Self>;
    }

    impl Handler<Display> for ProbeActor {
        type Result = ();

        fn handle(&mut self, msg: Display, _ctx: &mut Context<Self>) -> Self::Result {
            self.tx.send(msg.0).unwrap();
        }
    }

    #[test]
    fn stopping_the_system_before_run_skips_the_spawned_flow() {
        let (tx, rx) = mpsc::channel();
        let system = System::new();

        system.block_on(async move {
            let sum_addr = SumActor {}.start();
            let probe_addr = ProbeActor { tx }.start();

            Arbiter::current().spawn(async move {
                actix::clock::sleep(Duration::from_millis(10)).await;
                let sum = sum_addr.send(Value(6, 7)).await.unwrap();
                probe_addr.send(Display(sum)).await.unwrap();
            });

            System::current().stop();
        });

        system.run().unwrap();

        assert!(
            rx.recv_timeout(Duration::from_millis(50)).is_err(),
            "the spawned flow unexpectedly completed even though the system was stopped before run()",
        );
    }

    #[test]
    fn stopping_the_system_after_the_flow_completes_delivers_the_result() {
        let (tx, rx) = mpsc::channel();
        let system = System::new();

        system.block_on(async move {
            let sum_addr = SumActor {}.start();
            let probe_addr = ProbeActor { tx }.start();

            Arbiter::current().spawn(async move {
                actix::clock::sleep(Duration::from_millis(10)).await;
                let sum = sum_addr.send(Value(6, 7)).await.unwrap();
                probe_addr.send(Display(sum)).await.unwrap();
                System::current().stop();
            });
        });

        system.run().unwrap();

        assert_eq!(rx.recv_timeout(Duration::from_secs(1)).unwrap(), 13);
    }
}
