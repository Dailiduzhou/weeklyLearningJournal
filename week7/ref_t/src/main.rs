fn main() {
    let mut my = String::from("Hello world!");
    
    let word = first_word(&my);
    
    println!("{word}");

    println!("my: {my}");

    my.clear();
    println!("after clear: {my}");

    
}

fn first_word(s:&str) -> &str {
    let bytes = s.as_bytes();

    for (i, &item) in bytes.iter().enumerate() {
        if item == b' ' {
            return &s[..i]
        }
    }

    &s[..]
}
