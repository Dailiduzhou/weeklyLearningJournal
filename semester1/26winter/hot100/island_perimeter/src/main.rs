struct Solution;

impl Solution {
    pub fn island_perimeter(grid: Vec<Vec<i32>>) -> i32 {
        let r = grid.len();
        let c = grid[0].len();
        let mut perimeter = 0;

        for i in 0..r {
            for j in 0..c {
                if grid[i][j] == 1 {
                    perimeter += 4;
                    if i > 0 && grid[i - 1][j] == 1 {
                        perimeter -= 2;
                    }
                    if j > 0 && grid[i][j - 1] == 1 {
                        perimeter -= 2;
                    }
                }
            }
        }
        perimeter
    }
}
