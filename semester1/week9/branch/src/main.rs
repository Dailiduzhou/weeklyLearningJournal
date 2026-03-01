async fn action(input: Option<i32>) -> Option<String> {
    // 若 input（输入）是None，则返回 None
    // 事实上也可以这么写: `let i = input?;`
    let i = match input {
        Some(input) => input,
        None => return None,
    };

    Some(i.to_string())
}

#[tokio::main]
async fn main() {
    let (tx, mut rx) = tokio::sync::mpsc::channel(128);

    let mut done = false;
    let operation = action(None);
    tokio::pin!(operation);

    tokio::spawn(async move {
        let _ = tx.send(1).await;
        let _ = tx.send(3).await;
        let _ = tx.send(2).await;
    });

    loop {
        tokio::select! {
            res = &mut operation, if !done => {
                done = true;

                if let Some(v) = res {
                    println!("GOT = {}", v);
                    return;
                }
            }
            Some(v) = rx.recv() => {
                if v % 2 == 0 {
                    // `.set` 是 `Pin` 上定义的方法
                    operation.set(action(Some(v)));
                    done = false;
                }
            }
        }
    }
}