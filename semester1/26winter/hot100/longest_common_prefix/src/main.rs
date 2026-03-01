struct Solution;

impl Solution {
    pub fn longest_common_prefix(strs: Vec<String>) -> String {
        strs.into_iter()
            .reduce(|acc, cur| {
                acc.chars()
                    .zip(cur.chars())
                    .take_while(|(a, c)| a == c)
                    .map(|(c, _)| c)
                    .collect()
            })
            .unwrap()
    }
}

struct Solution1;

impl Solution1 {
    pub fn longest_common_prefix(strs: Vec<String>) -> String {
        strs.into_iter()
            .reduce(|mut acc, cur| {
                let len = acc
                    .chars()
                    .zip(cur.chars())
                    .take_while(|(a, c)| a == c)
                    .count();

                acc.truncate(len);
                acc
            })
            .unwrap()
    }
}

fn main() {
    let strs = vec![
        String::from("flower"),
        String::from("flow"),
        String::from("flight"),
    ];
    let strs1 = strs.clone();
    let res = Solution::longest_common_prefix(strs1);
    println!("{res}");

    let res = Solution1::longest_common_prefix(strs);
    println!("{res}");
}
