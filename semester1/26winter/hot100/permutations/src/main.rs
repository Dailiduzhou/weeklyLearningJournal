struct Solution;

impl Solution {
    pub fn permute(nums: Vec<i32>) -> Vec<Vec<i32>> {
        let mut used = [0; 6];
        let mut res = Vec::new();
        let mut per = Vec::new();

        fn bfs(
            per: &mut Vec<i32>,
            used: &mut [i32],
            nums: &Vec<i32>,
            res: &mut Vec<Vec<i32>>,
            now: usize,
        ) {
            if now == nums.len() {
                res.push(per.clone());
                return;
            }
            for i in 0..nums.len() {
                if used[i] == 1 {
                    continue;
                }

                per.push(nums[i]);
                used[i] = 1;
                bfs(per, used, nums, res, now + 1);
                per.pop();
                used[i] = 0;
            }
        }

        bfs(&mut per, &mut used, &nums, &mut res, 0);

        res
    }
}
