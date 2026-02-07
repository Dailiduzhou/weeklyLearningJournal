struct Solution;

impl Solution {
    pub fn fib(n: i32) -> i32 {
        (0..n)
            .fold((1, 0), |(prev1, prev2), _| (prev2, prev1 + prev2))
            .1
    }
}
