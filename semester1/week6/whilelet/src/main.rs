fn main() {
    let mut stack = Vec::new();

    stack.push(1);
    stack.push(2);
    stack.push(3);

    while let Some(top) = stack.pop() {
        println!("Now, top is {}", top)
    }


    // for (k, v) in stack.iter().enumerate() {
    //     println!("{} is at the index {}", k, v);
    // }
}
