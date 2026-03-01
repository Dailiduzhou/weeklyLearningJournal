use std::collections::{HashMap, HashSet};

struct Solution;

impl Solution {
    pub fn word_pattern(pattern: String, s: String) -> bool {
        if s.split_ascii_whitespace().count() != pattern.len() {
            return false;
        }

        let mut map: HashMap<u8, &str> = HashMap::<u8, &str>::new();
        let mut used: HashSet<&str> = HashSet::<&str>::new();

        for (k, v) in pattern.bytes().zip(s.split_ascii_whitespace()) {
            match map.insert(k, v) {
                Some(prev) if prev != v => return false,
                None if !used.insert(v) => return false,
                _ => continue,
            }
        }

        true
    }
}
