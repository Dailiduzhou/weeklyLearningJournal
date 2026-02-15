struct Solution;

impl Solution {
    pub fn max_area(height: Vec<i32>) -> i32 {
        let mut res = 0;
        let mut l = 0;
        let mut r = height.len() - 1;

        while l < r {
            res = res.max((r - l) as i32 * height[l].min(height[r]));
            if height[l] > height[r] && r > 0 {
                r -= 1;
            } else {
                l += 1;
            }
        }

        res
    }
}
