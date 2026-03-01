struct Solution;

impl Solution {
    pub fn combine(n: i32, k: i32) -> Vec<Vec<i32>> {
        let nums: Vec<i32> = (1..=n).collect();
        let mut res = Vec::new();
        let mut state = Vec::with_capacity(k as usize);
        Solution::backtrack(&nums, &mut state, k, 0, &mut res);
        res
    }

    fn backtrack(
        nums: &Vec<i32>,
        state: &mut Vec<i32>,
        k: i32,
        start: usize,
        res: &mut Vec<Vec<i32>>,
    ) {
        if state.len() == k as usize {
            res.push(state.clone());
            return;
        }
        for i in start..nums.len() {
            state.push(nums[i]);
            Solution::backtrack(nums, state, k, i + 1, res);
            state.pop();
        }
    }
}

struct Solution1;

impl Solution1 {
    pub fn combine(n: i32, k: i32) -> Vec<Vec<i32>> {
        let n = n as usize;
        let k = k as usize;
        let total = combination_count(n, k);
        let mut result = Vec::with_capacity(total);
        let mut current = Vec::with_capacity(k);
        backtrack(1, n, k, &mut current, &mut result);
        result
    }
}

fn combination_count(n: usize, k: usize) -> usize {
    if k > n {
        return 0;
    }
    let k = k.min(n - k);
    let mut res = 1;
    for i in 0..k {
        res = res * (n - i) / (i + 1);
    }
    res
}

fn backtrack(start: usize, n: usize, k: usize, current: &mut Vec<i32>, result: &mut Vec<Vec<i32>>) {
    if current.len() == k {
        result.push(current.clone());
        return;
    }
    for i in start..=n {
        if n - i + 1 < k - current.len() {
            break;
        }
        current.push(i as i32);
        backtrack(i + 1, n, k, current, result);
        current.pop();
    }
}
