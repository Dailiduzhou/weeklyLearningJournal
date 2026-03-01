struct Solution;

impl Solution {
    pub fn search_matrix(matrix: Vec<Vec<i32>>, target: i32) -> bool {
        use std::cmp::Ordering;
        let m = matrix.len();
        let n = matrix[0].len();
        let (mut low, mut high) = (0, m * n - 1);

        while low <= high {
            let mid = low + (high - low) / 2;
            let row = mid / n;
            let col = mid % n;
            let now = matrix[row][col];

            match now.cmp(&target) {
                Ordering::Equal => return true,
                Ordering::Less => low = mid + 1,
                Ordering::Greater => {
                    if mid == 0 {
                        return false;
                    }
                    high = mid - 1;
                }
            }
        }
        false
    }
}
