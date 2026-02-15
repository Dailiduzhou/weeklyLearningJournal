struct Solution;
use std::collections::HashMap;
const N_LETTERS: usize = (b'z' - b'a' + 1) as _;
impl Solution {
    pub fn group_anagrams(strs: Vec<String>) -> Vec<Vec<String>> {
        let mut map: HashMap<String, Vec<String>> = HashMap::new();

        for s in strs.iter() {
            let mut chars: Vec<char> = s.chars().collect();
            chars.sort_unstable();
            let key = chars.into_iter().collect::<String>();

            map.entry(key).or_default().push(s.clone());
            // map.entry(key).or_insert_with(Vec::new).push(s.clone());
        }

        map.into_values().collect()
    }

    pub fn group_anagrams1(strs: Vec<String>) -> Vec<Vec<String>> {
        strs.into_iter()
            .fold(
                HashMap::<[u8; N_LETTERS], Vec<String>>::new(),
                |mut map, s| {
                    let freqs = s.bytes().map(|b| (b - b'a') as usize).fold(
                        [0; N_LETTERS],
                        |mut freqs, bin| {
                            freqs[bin] += 1;
                            freqs
                        },
                    );
                    map.entry(freqs).or_default().push(s);
                    map
                },
            )
            .into_values()
            .collect()
    }
}
