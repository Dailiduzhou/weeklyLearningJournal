use criterion::{Criterion, criterion_group, criterion_main};
use rayon::prelude::*;
use std::hint::black_box;

fn expensive_computation(x: u64) -> u64 {
    (0..1000).fold(x, |acc, _| acc.wrapping_mul(7).wrapping_add(13))
}

fn bench_sum_comparison(c: &mut Criterion) {
    let data: Vec<u64> = (0..1_000_000).collect();

    let mut group = c.benchmark_group("sum_comparison");

    group.bench_function("sequential_sum", |b| {
        b.iter(|| {
            let sum: u64 = data.iter().map(|x| x * x).sum();
            black_box(sum)
        })
    });

    group.bench_function("parallel_sum", |b| {
        b.iter(|| {
            let sum: u64 = data.par_iter().map(|x| x * x).sum();
            black_box(sum)
        })
    });

    group.finish();
}

fn bench_sort_comparison(c: &mut Criterion) {
    let data: Vec<u64> = (0..100_000).rev().collect();

    let mut group = c.benchmark_group("sort_comparison");

    group.bench_function("sequential_sort", |b| {
        b.iter_batched(
            || data.clone(),
            |mut v| {
                v.sort();
                black_box(v)
            },
            criterion::BatchSize::SmallInput,
        )
    });

    group.bench_function("parallel_sort", |b| {
        b.iter_batched(
            || data.clone(),
            |mut v| {
                v.par_sort();
                black_box(v)
            },
            criterion::BatchSize::SmallInput,
        )
    });

    group.finish();
}

fn bench_filter_map_comparison(c: &mut Criterion) {
    let data: Vec<u64> = (0..100_000).collect();

    let mut group = c.benchmark_group("filter_map_comparison");

    group.bench_function("sequential_filter_map", |b| {
        b.iter(|| {
            let results: Vec<_> = data
                .iter()
                .filter(|&&x| x % 2 == 0)
                .map(|&x| expensive_computation(x))
                .collect();
            black_box(results)
        })
    });

    group.bench_function("parallel_filter_map", |b| {
        b.iter(|| {
            let results: Vec<_> = data
                .par_iter()
                .filter(|&&x| x % 2 == 0)
                .map(|&x| expensive_computation(x))
                .collect();
            black_box(results)
        })
    });

    group.finish();
}

criterion_group!(
    benches,
    bench_sum_comparison,
    bench_sort_comparison,
    bench_filter_map_comparison
);
criterion_main!(benches);
