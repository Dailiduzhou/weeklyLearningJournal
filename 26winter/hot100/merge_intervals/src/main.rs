struct Solution;

impl Solution {
    pub fn merge(mut intervals: Vec<Vec<i32>>) -> Vec<Vec<i32>> {
        let n = intervals.len();
        if n == 1 {
            return intervals;
        }

        intervals.sort_unstable_by_key(|interval| interval[0]);
        let mut res = Vec::with_capacity(n);
        let mut cur = vec![intervals[0][0], intervals[0][1]];

        for interval in intervals.into_iter().skip(1) {
            if cur[1] >= interval[0] {
                cur[1] = std::cmp::max(cur[1], interval[1]);
            } else {
                res.push(cur);
                cur = interval;
            }
        }

        res.push(cur);
        res
    }
}
