use actix::prelude::*;

#[derive(Message)]
#[rtype(result = "()")]
struct OrderShipped(usize);

#[derive(Message)]
#[rtype(result = "()")]
struct Ship(usize);

/// Subscribe to order shipped event.
#[derive(Message)]
#[rtype(result = "()")]
struct Subscribe(pub Recipient<OrderShipped>);

/// Actor that provides order shipped event subscriptions
struct OrderEvents {
    subscribers: Vec<Recipient<OrderShipped>>,
}

impl OrderEvents {
    fn new() -> Self {
        OrderEvents {
            subscribers: vec![],
        }
    }
}

impl Actor for OrderEvents {
    type Context = Context<Self>;
}

impl OrderEvents {
    /// Send event to all subscribers
    fn notify(&mut self, order_id: usize) {
        for subscr in &self.subscribers {
            subscr.do_send(OrderShipped(order_id));
        }
    }
}

/// Subscribe to shipment event
impl Handler<Subscribe> for OrderEvents {
    type Result = ();

    fn handle(&mut self, msg: Subscribe, _: &mut Self::Context) {
        self.subscribers.push(msg.0);
    }
}

/// Subscribe to ship message
impl Handler<Ship> for OrderEvents {
    type Result = ();

    fn handle(&mut self, msg: Ship, _ctx: &mut Self::Context) {
        self.notify(msg.0);
        System::current().stop();
    }
}

/// Email Subscriber
struct EmailSubscriber;

impl Actor for EmailSubscriber {
    type Context = Context<Self>;
}

impl Handler<OrderShipped> for EmailSubscriber {
    type Result = ();

    fn handle(&mut self, msg: OrderShipped, _ctx: &mut Self::Context) {
        println!("Email sent for order {}", msg.0)
    }
}

/// SMS Subscriber
struct SmsSubscriber;

impl Actor for SmsSubscriber {
    type Context = Context<Self>;
}

impl Handler<OrderShipped> for SmsSubscriber {
    type Result = ();

    fn handle(&mut self, msg: OrderShipped, _ctx: &mut Self::Context) {
        println!("SMS sent for order {}", msg.0)
    }
}

#[actix::main]
async fn main() -> Result<(), actix::MailboxError> {
    let email_subscriber = Subscribe(EmailSubscriber {}.start().recipient());
    let sms_subscriber = Subscribe(SmsSubscriber {}.start().recipient());
    let order_event = OrderEvents::new().start();

    order_event.send(email_subscriber).await?;
    order_event.send(sms_subscriber).await?;
    order_event.send(Ship(1)).await?;
    order_event.send(Ship(3)).await?;

    Ok(())
}
