struct Solution;

impl Solution {
    pub fn fizz_buzz(n: i32) -> Vec<String> {
        let mut res = Vec::with_capacity(n as usize);
        let mut cnt = 1;
        while cnt <= n {
            if cnt % 15 == 0 {
                res.push("FizzBuzz".to_string());
            } else if cnt % 5 == 0 {
                res.push("Buzz".to_string());
            } else if cnt % 3 == 0 {
                res.push("Fizz".to_string());
            } else {
                res.push(format!("{}", cnt));
            }

            cnt += 1;
        }

        res
    }
}
