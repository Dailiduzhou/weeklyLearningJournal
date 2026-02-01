struct Solution;

impl Solution {
    pub fn trap(height: Vec<i32>) -> i32 {
        use std::cmp::{max, min};
        let mut i = 0_usize;
        let mut j = height.len() - 1;
        let mut res = 0;

        loop {
            if i > j {
                break;
            }

            let cap = min(height[i], height[j]) * (j - i) as i32;
            res = max(cap, res);
            if height[i] < height[j] {
                i += 1;
            } else {
                j -= 1;
            }
        }
        res
    }
}

fn main() {
    let height = vec![4, 2, 0, 3, 2, 5];
    let res = Solution::trap(height);
    println!("{res}");
}
