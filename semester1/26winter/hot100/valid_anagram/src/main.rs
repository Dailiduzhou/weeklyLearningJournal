use std::collections::HashMap;

struct Solution;

impl Solution {
    pub fn is_anagram(s: String, t: String) -> bool {
        if s.len() != t.len() {
            return false;
        }
        let mut map = std::collections::HashMap::new();
        s.chars().for_each(|c| *map.entry(c).or_insert(0) += 1);
        t.chars().for_each(|c| *map.entry(c).or_insert(0) -= 1);
        map.into_values().all(|v| v == 0)
    }
}

struct Solution1;

impl Solution1 {
    pub fn is_anagram(s: String, t: String) -> bool {
        let mut map: HashMap<char, i32> = std::collections::HashMap::new();
        s.chars().for_each(|c| *map.entry(c).or_insert(0) += 1);
        for c in t.chars() {
            if *map.get(&c).unwrap_or(&0) == 0 {
                return false;
            }

            *map.entry(c).or_insert(0) -= 1;
        }
        true
    }
}
