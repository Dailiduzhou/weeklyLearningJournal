struct Solution;

impl Solution {
    pub fn two_sum(nums: Vec<i32>, target: i32) -> Vec<i32> {
        use std::collections::HashMap;

        let mut map = HashMap::new();

        for (i, &num) in nums.iter().enumerate() {
            match map.get(&(target - num)) {
                Some(v) => return vec![(*v), i as i32],
                None => map.insert(num, i as i32),
            };
        }
        vec![]
    }
}

fn main() {
    let nums = vec![2, 7, 11, 15];
    let target = 9;
    let res = Solution::two_sum(nums, target);
    println!("{:?}", res);
}
