struct Solution;

impl Solution {
    pub fn exist(mut board: Vec<Vec<char>>, word: String) -> bool {
        let m = board.len();
        let n = board[0].len();
        let word_chars: Vec<char> = word.chars().collect();

        for i in 0..m {
            for j in 0..n {
                if backtrack(&mut board, i as i32, j as i32, 0, &word_chars) {
                    return true;
                }
            }
        }
        false
    }
}

fn backtrack(board: &mut Vec<Vec<char>>, i: i32, j: i32, k: usize, word: &[char]) -> bool {
    if k == word.len() {
        return true;
    }
    if i < 0
        || i >= board.len() as i32
        || j < 0
        || j >= board[0].len() as i32
        || board[i as usize][j as usize] != word[k]
    {
        return false;
    }

    let temp = board[i as usize][j as usize];
    board[i as usize][j as usize] = '\0';

    if backtrack(board, i + 1, j, k + 1, word)
        || backtrack(board, i - 1, j, k + 1, word)
        || backtrack(board, i, j + 1, k + 1, word)
        || backtrack(board, i, j - 1, k + 1, word)
    {
        return true;
    }

    board[i as usize][j as usize] = temp;
    false
}
