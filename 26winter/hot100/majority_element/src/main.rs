struct Solution;

impl Solution {
    pub fn majority_element(nums: Vec<i32>) -> i32 {
        let mut candidate = None;
        let mut count = 0;

        for num in nums {
            if count == 0 {
                candidate = Some(num)
            }

            count += if candidate == Some(num) { 1 } else { -1 };
        }

        candidate.unwrap()
    }
}
