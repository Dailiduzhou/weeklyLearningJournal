use std::collections::HashSet;

struct Solution;

impl Solution {
    pub fn contains_duplicate(nums: Vec<i32>) -> bool {
        let mut exits = HashSet::new();
        !nums.into_iter().all(|x| exits.insert(x))
    }

    pub fn contains_duplicate1(nums: Vec<i32>) -> bool {
        nums.len() != HashSet::<i32>::from_iter(nums).len()
    }
}
