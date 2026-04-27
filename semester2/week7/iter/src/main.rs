fn main() {
    let data = [1, 2, 3, 6, 7, 8];
    fn fun_name(x: &&i32) -> bool {
        *x > &5
    }
    let _ = data.iter().filter(fun_name).map(|x| {
        println!("{x}");
    });
}
