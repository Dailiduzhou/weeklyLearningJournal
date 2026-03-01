struct Solution;

impl Solution {
    pub fn is_palindrome_iter(s: String) -> bool {
        let iter = s
            .chars()
            .filter(|c| c.is_alphanumeric())
            .map(|c| c.to_ascii_lowercase());

        iter.clone().eq(iter.rev())
    }

    pub fn is_palindrome(s: String) -> bool {
        let chars: Vec<char> = s
            .chars()
            .filter(|c| c.is_alphanumeric())
            .map(|c| c.to_ascii_lowercase())
            .collect();

        if chars.is_empty() {
            return true;
        }

        let (mut left, mut right) = (0, chars.len() - 1);

        while left < right {
            if chars[left] != chars[right] {
                return false;
            }
            left += 1;
            right -= 1;
        }

        true
    }
}
