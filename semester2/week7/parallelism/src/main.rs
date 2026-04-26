use rayon::prelude::*;

fn main() {
    let data: Vec<u64> = (0..1_000_000).collect();

    // Sequential:
    let sum_seq: u64 = data.iter().map(|x| x * x).sum();

    // Parallel — just change .iter() to .par_iter():
    let sum_par: u64 = data.par_iter().map(|x| x * x).sum();

    assert_eq!(sum_seq, sum_par);

    // Parallel sort:
    let mut numbers = vec![5, 2, 8, 1, 9, 3];
    numbers.par_sort();

    println!("{:?}", numbers);

    // Parallel processing with map/filter/collect:
    let results: Vec<_> = data
        .par_iter()
        .filter(|&&x| x % 2 == 0)
        .map(|&x| expensive_computation(x))
        .collect();

    drop(results);
}

fn expensive_computation(x: u64) -> u64 {
    (0..1000).fold(x, |acc, _| acc.wrapping_mul(7).wrapping_add(13))
}
