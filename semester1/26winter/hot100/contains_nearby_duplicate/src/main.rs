use core::num;
use std::collections::HashMap;

struct Solution;

impl Solution {
    pub fn contains_nearby_duplicate(nums: Vec<i32>, k: i32) -> bool {
        let mut map = HashMap::new();

        for (i, &num) in nums.iter().enumerate() {
            if let Some(&prev) = map.get(&num)
                && i as i32 - prev <= k
            {
                return true;
            }
            map.insert(num, i as i32);
        }
        false
    }
}
