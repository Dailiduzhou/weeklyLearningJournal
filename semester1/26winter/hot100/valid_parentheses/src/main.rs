struct Solution;

impl Solution {
    pub fn is_valid(s: String) -> bool {
        s.chars()
            .try_fold(vec![], |mut stack, c| {
                match c {
                    '(' => stack.push(')'),
                    '[' => stack.push(']'),
                    '{' => stack.push('}'),
                    other => {
                        if Some(other) != stack.pop() {
                            return Err(());
                        }
                    }
                }

                Ok(stack)
            })
            .map(|stack| stack.is_empty())
            .unwrap_or(false)
    }
}

fn main() {
    let s = String::from("([]})");
    let res = Solution::is_valid(s);
    println!("{res}");
}
