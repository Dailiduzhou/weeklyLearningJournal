fn is_safe(row: i32, col: i32, cols: &[i32]) -> bool {
    for i in 0..row {
        if cols[i as usize] == col || (cols[i as usize] - col).abs() == (i - row).abs() {
            return false;
        }
    }
    true
}

fn new_row(col: i32, n: i32) -> Vec<char> {
    let mut new_vec = vec!['.'; n as usize];
    new_vec[col as usize] = 'Q';
    new_vec
}

fn backtrack(
    row: i32,
    n: i32,
    state: &mut Vec<Vec<char>>,
    cols: &mut Vec<i32>,
    res: &mut Vec<Vec<Vec<char>>>,
) {
    if row == n {
        res.push(state.clone());
        return;
    }

    for col in 0..n {
        if is_safe(row, col, cols) {
            cols[row as usize] = col; // 记录当前行皇后所在的列
            state.push(new_row(col, n));
            backtrack(row + 1, n, state, cols, res);
            cols[row as usize] = -1; // 回溯
            state.pop();
        }
    }
}

fn main() {
    let n = 10;
    let mut cols = vec![-1; n as usize]; // 初始化所有行都未放置皇后
    let mut state = Vec::new(); // 当前棋盘状态
    let mut res = Vec::new(); // 存储所有解

    backtrack(0, n, &mut state, &mut cols, &mut res);

    println!("共有 {} 种解法\n", res.len());

    for (idx, board) in res.iter().enumerate() {
        println!("解法 {}:", idx + 1);
        for row in board {
            for col in row {
                print!("{col} ");
            }
            println!();
        }
        println!();
    }
}
