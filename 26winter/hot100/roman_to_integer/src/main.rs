use std::collections::HashMap;

struct Solution;

impl Solution {
    pub fn roman_to_int(s: String) -> i32 {
        let pair = vec![
            ('I', 1),
            ('V', 5),
            ('X', 10),
            ('L', 50),
            ('C', 100),
            ('D', 500),
            ('M', 1000),
        ];

        let map: HashMap<char, i32> = pair.into_iter().collect();
        let chars: Vec<char> = s.chars().collect();
        let n = s.len();
        let mut res = map[&chars[n - 1]];
        for i in (0..n - 1).rev() {
            if map[&chars[i]] < map[&chars[i + 1]] {
                res -= map[&chars[i]];
            } else {
                res += map[&chars[i]];
            }
        }
        res
    }
}

fn main() {
    let s: String = String::from("LVIII");
    let res = Solution::roman_to_int(s);
    println!("{res}");
}
