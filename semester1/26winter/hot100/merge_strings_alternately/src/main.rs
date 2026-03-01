struct Solution;

impl Solution {
    pub fn merge_alternately(word1: String, word2: String) -> String {
        let mut res = String::with_capacity(word1.len() + word2.len());
        let mut i1 = word1.chars();
        let mut i2 = word2.chars();

        loop {
            match (i1.next(), i2.next()) {
                (Some(c1), Some(c2)) => {
                    res.push(c1);
                    res.push(c2);
                }
                (Some(c1), None) => {
                    res.push(c1);
                    res.extend(i1);
                    break;
                }
                (None, Some(c2)) => {
                    res.push(c2);
                    res.extend(i2);
                    break;
                }
                (None, None) => break,
            }
        }
        res
    }
}

fn main() {
    let res = Solution::merge_alternately(String::from("abcd"), String::from("pq"));
    println!("{res}");
}
