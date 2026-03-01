mod my {
    pub struct Rectangle {
        width: u32,
        pub height: u32,
    }

    impl Rectangle {
        pub fn new(width: u32, height: u32) -> Self {
            Rectangle { width, height }
        }
        pub fn width(&self) -> u32 {
            return self.width;
        }
        pub fn height(&self) -> u32 {
            return self.height;
        }
    }
}

fn main() {
    let rect1 = my::Rectangle::new(30, 50);

    println!("{}", rect1.width()); // OK
    println!("{}", rect1.height()); // OK
    // println!("{}", rect1.width); // Error - the visibility of field defaults to private
    println!("{}", rect1.height); // OK
}