struct Solution;

impl Solution {
    pub fn search(nums: Vec<i32>, target: i32) -> i32 {
        use std::cmp::Ordering;
        let mut i = 0_usize;
        let mut j = nums.len() - 1;
        while i < j {
            let med = i + (j - i) / 2;
            match target.cmp(&nums[med]) {
                Ordering::Greater => i = med + 1,
                Ordering::Less => j = med,
                Ordering::Equal => {
                    return med as i32;
                }
            }
        }
        -1
    }
}

fn main() {
    let nums = vec![-1, 0, 3, 5, 9, 12];
    let res = Solution::search(nums, 9);
    println!("{res}");

    let nums = vec![-1, 0, 3, 5, 9, 12];
    let res = Solution::search(nums, 2);
    println!("{res}");
}
