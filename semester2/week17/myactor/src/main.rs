use myactor::{ActorError, MyActorHandle};

#[tokio::main]
async fn main() -> Result<(), ActorError> {
    let actor_handle = MyActorHandle::new();

    let first = actor_handle.incr_counter().await?;
    assert_eq!(first, 1);

    let second = actor_handle.incr_counter_by(32).await?;
    assert_eq!(second, 33);
    println!("counter after increments: {second}");

    Ok(())
}
