struct Solution;

impl Solution {
    pub fn missing_number(nums: Vec<i32>) -> i32 {
        (1..=nums.len()).sum::<usize>() as i32 - nums.into_iter().sum::<i32>()
    }

    pub fn missing_number1(nums: Vec<i32>) -> i32 {
        nums.into_iter()
            .enumerate()
            .fold(0, |acc, (i, x)| acc ^ (i + 1) as i32 ^ x)
    }
}
