struct Solution;

use std::collections::HashSet;

impl Solution {
    pub fn longest_consecutive(nums: Vec<i32>) -> i32 {
        if nums.is_empty() {
            return 0;
        }

        let set: HashSet<i32> = nums.into_iter().collect();
        let mut longest = 1;

        for &x in set.iter() {
            if !set.contains(&(x - 1)) {
                let mut curr = x;
                let mut length = 1;

                while set.contains(&(curr + 1)) {
                    curr += 1;
                    length += 1;
                }
                longest = longest.max(length);
            }
        }
        longest
    }
}

struct Solution1;

impl Solution1 {
    pub fn longest_consecutive(nums: Vec<i32>) -> i32 {
        let num_set: HashSet<i32> = nums.into_iter().collect();
        let mut ans = 0;
        for &num in &num_set {
            if !num_set.contains(&(num - 1)) {
                let count = (num..).take_while(|x| num_set.contains(x)).count();
                ans = ans.max(count);
            }
        }
        ans as i32
    }
}
