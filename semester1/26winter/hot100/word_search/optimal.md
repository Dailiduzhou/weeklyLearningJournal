# A optimal Solution

```rust
impl Solution {
    /// 四个方向的偏移量：上、右、下、左
    const POSSIBLE_SIDES: [(i32, i32); 4] = [(-1, 0), (0, 1), (1, 0), (0, -1)];

    /// 判断 board 中是否存在一条路径可以拼出 word（LeetCode 79 - Word Search）。
    ///
    /// 优化策略：
    /// 1. 如果 board 总格子数不够，直接返回 false。
    /// 2. 统计 board 中各字符的出现次数，若 word 中任一字符需求量超过 board 供给量，直接返回 false。
    /// 3. 比较 word 首尾字符在 board 中的出现频率，从频率更低的一端开始搜索，以减少 DFS 分支数。
    pub fn exist(mut board: Vec<Vec<char>>, mut word: String) -> bool {
        let (m, n) = (board.len(), board[0].len());

        // 剪枝：board 格子总数不足以容纳 word
        let total = m * n;
        if total < word.len() {
            return false;
        }

        // 统计 board 中每个字符的出现次数（利用 ASCII 值作为下标）
        let mut board_counts: [usize; 128] = [0; 128];
        for i in 0..m {
            for j in 0..n {
                board_counts[board[i][j] as usize] += 1;
            }
        }

        let first_char = word.chars().next().unwrap();
        let last_char = word.chars().last().unwrap();
        let first_count = board_counts[first_char as usize];
        let last_count = board_counts[last_char as usize];

        // 统计 word 中每个字符的出现次数
        let mut word_counts: [usize; 128] = [0; 128];
        for letter in word.chars() {
            word_counts[letter as usize] += 1;
        }

        // 剪枝：word 中某字符的需求量超过 board 中的供给量
        for i in 0..128 {
            if word_counts[i] > board_counts[i] {
                return false;
            }
        }

        // 优化：若首字符在 board 中出现次数多于末字符，则将 word 反转，
        // 从出现频率更低的字符端开始搜索，可以更快地排除无效路径。
        if first_count > last_count {
            word = word.chars().rev().collect::<String>();
        }

        // 遍历 board 中每个格子作为起点，尝试 DFS 匹配
        for i in 0..m {
            for j in 0..n {
                if Self::dfs(&word, &mut board, (i, j)) {
                    return true;
                }
            }
        }
        false
    }

    /// 统计 board 中指定字符 letter 的出现次数（本解法中未使用，已由 board_counts 数组替代）。
    #[allow(dead_code)]
    pub fn count(board: &[Vec<char>], letter: char) -> i32 {
        let (m, n) = (board.len(), board[0].len());
        let mut count = 0;
        for i in 0..m {
            for j in 0..n {
                if board[i][j] == letter {
                    count += 1;
                }
            }
        }
        count
    }

    /// DFS 回溯搜索：尝试从 board[i][j] 开始匹配 word 的剩余部分。
    ///
    /// - 使用 `#` 作为标记表示当前格子已被访问，防止重复使用。
    /// - 递归结束后恢复格子原始值（回溯）。
    pub fn dfs(word: &str, board: &mut Vec<Vec<char>>, (i, j): (usize, usize)) -> bool {
        let current = board[i][j];

        // 如果 word 只剩一个字符，且当前格子匹配，则搜索成功
        if word.len() == 1 && current == word.chars().next().unwrap() {
            return true;
        }

        // 当前格子不匹配 word 首字符，或已被访问（标记为 '#'），则剪枝
        if current != word.chars().next().unwrap() || current == '#' {
            return false;
        }

        // 收集当前位置的所有合法邻居坐标（不越界）
        let mut valid_points: Vec<(usize, usize)> = Vec::new();
        for &(x, y) in Self::POSSIBLE_SIDES.iter() {
            let (new_i, new_j) = (i as i32 + x, j as i32 + y);
            if new_i >= 0
                && new_i < board.len() as i32
                && new_j >= 0
                && new_j < board[0].len() as i32
            {
                valid_points.push((new_i as usize, new_j as usize));
            }
        }

        // 标记当前格子为已访问
        board[i][j] = '#';

        // 向四个方向递归搜索 word 的下一个字符
        for side in valid_points {
            if Self::dfs(&word[1..], board, side) {
                return true;
            }
        }

        // 回溯：恢复当前格子的原始值
        board[i][j] = current;
        false
    }
}
```
