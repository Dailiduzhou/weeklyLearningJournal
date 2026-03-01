struct Solution;

impl Solution {
    pub fn is_valid_sudoku(board: Vec<Vec<char>>) -> bool {
        let mut set = [0_u32; 9]; // to track visited places for each digit

        for row in 0..9 {
            for col in 0..9 {
                if board[row][col] == '.' {
                    continue;
                }

                let row_bits = 1 << row;
                let col_bits = 1 << 9 << col;
                let cell = row / 3 * 3 + col / 3;
                let cell_bits = 1 << 18 << cell;

                let d = (board[row][col] as u8 - b'1') as usize;
                if set[d] & row_bits != 0 || set[d] & col_bits != 0 || set[d] & cell_bits != 0 {
                    return false;
                }

                set[d] |= row_bits;
                set[d] |= col_bits;
                set[d] |= cell_bits;
            }
        }

        true
    }
}
