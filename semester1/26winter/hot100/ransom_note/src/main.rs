struct Solution;

impl Solution {
    pub fn can_construct(ransom_note: String, magazine: String) -> bool {
        let mut map: [i32; 26] = [0; 26];

        for c in magazine.chars() {
            map[c as usize - 'a' as usize] += 1;
        }

        for c in ransom_note.chars() {
            if map[c as usize - 'a' as usize] == 0 {
                return false;
            }

            map[c as usize - 'a' as usize] -= 1;
        }
        true
    }

    pub fn can_construct_hashmap(ransom_note: String, magazine: String) -> bool {
        let mut map: std::collections::HashMap<char, i32> = std::collections::HashMap::new();

        for c in magazine.chars() {
            map.entry(c).and_modify(|count| *count += 1).or_insert(1);
        }

        for c in ransom_note.chars() {
            match map.get_mut(&c) {
                Some(count) => *count -= 1,
                None => return false,
            }
        }
        true
    }
}
