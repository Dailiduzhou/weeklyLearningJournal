struct Solution;

impl Solution {
    pub fn cal_points(operations: Vec<String>) -> i32 {
        let mut res: Vec<i32> = Vec::new();
        for x in operations {
            match x.as_str() {
                "+" => {
                    if res.len() >= 2 {
                        let l = res.len();
                        let sum = res[l - 1] + res[l - 2];
                        res.push(sum);
                    }
                }
                "D" => {
                    if let Some(&last) = res.last() {
                        res.push(last * 2);
                    }
                }
                "C" => {
                    res.pop();
                }
                other => {
                    if let Ok(n) = other.parse::<i32>() {
                        res.push(n);
                    }
                }
            }
        }
        res.iter().sum()
    }
}
