struct Solution;

impl Solution {
    pub fn search(nums: Vec<i32>, target: i32) -> i32 {
        let f = |x: &i32| (*x < nums[0], *x);
        match nums.binary_search_by_key(&f(&target), f) {
            Ok(i) => i as i32,
            Err(_) => -1,
        }
    }

    pub fn search_normal(nums: Vec<i32>, target: i32) -> i32 {
        let (mut l, mut r) = (0, nums.len() - 1);

        while l < r {
            let mid = l + (r - l) / 2;

            if nums[mid] == target {
                return mid as i32;
            }

            if nums[l] <= nums[mid] {
                if nums[l] <= target && target < nums[mid] {
                    r = mid - 1;
                } else {
                    l = mid + 1;
                }
            } else if nums[mid] < target && target <= nums[r] {
                l = mid + 1;
            } else {
                r = mid - 1;
            }
        }

        -1
    }
}
