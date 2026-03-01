struct Solution;

impl Solution {
    pub fn contains_nearby_duplicate(nums: Vec<i32>, k: i32) -> bool {
        use std::collections::HashMap;
        let mut map: HashMap<i32, i32> = HashMap::new();

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
