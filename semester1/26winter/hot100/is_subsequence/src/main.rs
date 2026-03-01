struct Solution;

impl Solution {
    pub fn is_subsequence(s: String, t: String) -> bool {
        let v1: Vec<char> = s.chars().collect();
        let v2: Vec<char> = t.chars().collect();

        let mut i = 0;
        let mut j = 0;

        while i < v1.len() && j < v2.len() {
            if v1[i] == v2[j] {
                i += 1;
            }
            j += 1
        }

        i == v1.len()
    }
}
