struct Solution;

impl Solution {
    pub fn three_sum(mut nums: Vec<i32>) -> Vec<Vec<i32>> {
        use std::cmp::Ordering;
        let n = nums.len();
        let mut res = Vec::new();

        nums.sort_unstable();

        if nums[0] + nums[1] + nums[2] > 0 || nums[n - 1] + nums[n - 2] + nums[n - 3] < 0 {
            return res;
        }

        for i in 0..n - 2 {
            let x = nums[i];

            if i > 0 && x == nums[i - 1] {
                continue;
            }

            if x + nums[i + 1] + nums[i + 2] > 0 {
                break;
            }

            if x + nums[n - 2] + nums[n - 1] < 0 {
                continue;
            }

            let mut l = i + 1;
            let mut r = n - 1;

            while l < r {
                let sum = x + nums[l] + nums[r];
                match sum.cmp(&0) {
                    Ordering::Equal => {
                        res.push(vec![x, nums[l], nums[r]]);
                        l += 1;
                        r -= 1;

                        while l < r && nums[l] == nums[l - 1] {
                            l += 1;
                        }
                        while l < r && nums[r] == nums[r + 1] {
                            r -= 1;
                        }
                    }
                    Ordering::Less => l += 1,
                    Ordering::Greater => r -= 1,
                }
            }
        }
        res
    }
}
