fn main() {
    let s = String::from("hello world");

    let hel = &s[0..5];
    let wor = &s[6..11];

    print!("part1:{}\npart2:{}", hel, wor);
}
