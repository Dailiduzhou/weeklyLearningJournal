fn main() {
    let mut v = vec![1, 2, 3, 4, 5];

    let first = v[0];
    println!("The value of first is {first}");

    v.push(6);

    match v.get(0){
        Some(x) => println!("The first element: {x}"),
        None => println!("There is no such element"),
    }

    let mut v1: Vec<i32> = vec![1, 3, 4];
    let a: i32 = v1[1];

    for i in &v1 {
        println!("{i}");
    }

    for i in &mut v1 {
        *i += 20;
        println!("{}", *i);
    }

    println!("{a}");
}
