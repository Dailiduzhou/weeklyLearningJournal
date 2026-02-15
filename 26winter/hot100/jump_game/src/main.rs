struct Solution;

impl Solution {
    pub fn can_jump(nums: Vec<i32>) -> bool {
        let n = nums.len();

        if n == 1 || (n == 2 && nums[0] >= 1) {
            return true;
        }
        let mut goal = n - 1;

        for i in (0..=n - 2).rev() {
            if i + nums[i] as usize >= goal {
                goal = i;
            }
        }

        goal == 0
    }
}
