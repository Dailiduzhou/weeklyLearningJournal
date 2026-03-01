struct Solution;

impl Solution {
    pub fn trap(height: Vec<i32>) -> i32 {
        let mut stack: Vec<usize> = Vec::new();
        let mut res: i32 = 0;
        for i in 0..height.len() {
            while !stack.is_empty() && height[*stack.last().unwrap()] <= height[i] {
                let top = stack.pop().unwrap();
                if stack.is_empty() {
                    break;
                }
                let left = *stack.last().unwrap();
                let w: i32 = (i - left - 1) as i32;
                let h = height[left].min(height[i]) - height[top];
                res += w * h;
            }
            stack.push(i);
        }
        res
    }
}
