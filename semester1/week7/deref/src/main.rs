use std::ops::Deref;

#[derive(Debug)]
struct MyBox<T>(T);

impl<T> MyBox<T>{
    fn new(x: T) -> MyBox<T> {
        MyBox(x)
    }
}

impl<T> Deref for MyBox<T> {
    type Target = T;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}




fn main() {
    let bb = MyBox::new(String::from("innocent"));
    let s = String::from("innocent");
    let _b = (*bb).clone();
    display(&(*bb));
    display(&s);
}

fn display(s: &str) {
    println!("{}", s);
}
