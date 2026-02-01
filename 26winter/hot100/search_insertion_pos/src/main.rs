struct Solution;

impl Solution {
    pub fn search_insert(nums: Vec<i32>, target: i32) -> i32 {
        match nums.binary_search(&target) {
            Ok(x) => x as i32,
            Err(x) => x as i32,
        }
    }
}

fn main() {
    let nums = vec![1, 3, 5, 6];
    let res = Solution::search_insert(nums.clone(), 5);
    println!("{res}");

    let res = Solution::search_insert(nums, 3);
    println!("{res}");
}
