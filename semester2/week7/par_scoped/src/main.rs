fn parallel_map<T: Sync, R: Send>(data: &[T], f: fn(&T) -> R, num_threads: usize) -> Vec<R> {
    let chunk_size = data.len().div_ceil(num_threads);
    let mut results = Vec::with_capacity(data.len());

    std::thread::scope(|s| {
        let mut handles = Vec::new();
        for chunk in data.chunks(chunk_size) {
            handles.push(s.spawn(move || chunk.iter().map(f).collect::<Vec<_>>()))
        }
        for h in handles {
            results.extend(h.join().unwrap());
        }
    });

    results
}

fn main() {
    let data: Vec<_> = (1..=20).collect();
    let squres = parallel_map(&data, |x| x * x, 4);
    println!("Parallel squres: {squres:?}");
}
