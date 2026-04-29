use rand::random;

fn main() {
    let samples: Vec<f64> = std::iter::from_fn(|| Some(random()))
        .take(10)
        .take_while(|_| !random::<u8>().is_multiple_of(3))
        .collect();
    println!("{samples:?}");
}
