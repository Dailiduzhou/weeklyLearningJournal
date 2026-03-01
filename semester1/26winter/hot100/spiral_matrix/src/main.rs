struct Solution;

impl Solution {
    pub fn spiral_order(matrix: Vec<Vec<i32>>) -> Vec<i32> {
        enum Dire {
            Up,
            Down,
            Left,
            Right,
        }
        let mut direction = Dire::Right;
        let m = matrix.len();
        let n = matrix[0].len();
        let tot = m * n;
        let mut res = Vec::with_capacity(tot);
        let mut round = 0;
        let mut x = 0;
        let mut y = 0;
        for _ in 0..tot {
            res.push(matrix[x][y]);

            match direction {
                Dire::Up => {
                    if x == round + 1 {
                        y += 1;
                        direction = Dire::Right;
                        round += 1;
                    } else {
                        x -= 1;
                    }
                }
                Dire::Down => {
                    if x + 1 == m - round {
                        y -= 1;
                        direction = Dire::Left;
                    } else {
                        x += 1;
                    }
                }
                Dire::Left => {
                    if y == round {
                        x -= 1;
                        direction = Dire::Up;
                    } else {
                        y -= 1;
                    }
                }
                Dire::Right => {
                    if y + 1 == n - round {
                        x += 1;
                        direction = Dire::Down;
                    } else {
                        y += 1;
                    }
                }
            }
        }
        res
    }
}

fn main() {
    let matri = vec![vec![1, 2, 3], vec![4, 5, 6], vec![7, 8, 9]];
    let res = Solution::spiral_order(matri);
    for i in res {
        print!("{} ", i);
    }
}
