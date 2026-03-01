struct Solution;

impl Solution {
    pub fn combination_sum(mut candidates: Vec<i32>, target: i32) -> Vec<Vec<i32>> {
        let mut state: Vec<i32> = Vec::new();
        candidates.sort_unstable();
        let mut res = vec![];
        Solution::backtrack(&mut state, target, &candidates, 0, &mut res);

        res
    }

    pub fn backtrack(
        state: &mut Vec<i32>,
        target: i32,
        choices: &Vec<i32>,
        start: usize,
        res: &mut Vec<Vec<i32>>,
    ) {
        if target == 0 {
            res.push(state.clone());
            return;
        }

        for i in start..choices.len() {
            if target - choices[i] < 0 {
                break;
            }

            state.push(choices[i]);
            Solution::backtrack(state, target - choices[i], choices, i, res);
            state.pop();
        }
    }
}
