struct Solution;

impl Solution {
    pub fn length_of_longest_substring(s: String) -> i32 {
        use std::collections::HashSet;
        let s: Vec<char> = s.chars().collect();
        let mut l = 0;
        let mut maxl = 0;
        let mut set = HashSet::new();

        for r in 0..s.len() {
            while set.contains(&s[r]) {
                set.remove(&s[l]);
                l += 1;
            }

            set.insert(s[r]);
            maxl = maxl.max(r - l + 1);
        }

        maxl as i32
    }

    pub fn length_of_longest_substring1(s: String) -> i32 {
        let mut max_length = 0;
        let mut last_index = [0; 128];
        let mut start = 0;

        for (end, ch) in s.chars().enumerate() {
            start = start.max(last_index[ch as usize]);
            max_length = max_length.max(end - start + 1);
            last_index[ch as usize] = end + 1;
        }

        max_length as i32
    }
}
