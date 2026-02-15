struct Solution;

impl Solution {
    pub fn find_closest_number(nums: Vec<i32>) -> i32 {
        let mut min = i32::MAX;
        for n in nums {
            if n.abs() < min.abs() || (n.abs() == min.abs() && n >= 0) {
                min = n;
            }
        }
        min
    }
}
