struct Solution;

impl Solution {
    pub fn flood_fill(image: Vec<Vec<i32>>, sr: i32, sc: i32, color: i32) -> Vec<Vec<i32>> {
        let mut image: Vec<Vec<i32>> = image;
        let rows = image.len();
        let cols = image[0].len();
        let start_color = image[sr as usize][sc as usize];

        if start_color == color {
            return image;
        }
        let process_pixel = |i: usize, j: usize, image: &mut Vec<Vec<i32>>| {
            if image[i][j] == start_color {
                image[i][j] = color;
                true
            } else {
                false
            }
        };
        let mut stack = vec![(sr as usize, sc as usize)];

        while let Some((i, j)) = stack.pop() {
            if process_pixel(i, j, &mut image) {
                if i > 0 {
                    stack.push((i - 1, j));
                }
                if i + 1 < rows {
                    stack.push((i + 1, j));
                }
                if j > 0 {
                    stack.push((i, j - 1));
                }
                if j + 1 < cols {
                    stack.push((i, j + 1));
                }
            }
        }

        image
    }
}
