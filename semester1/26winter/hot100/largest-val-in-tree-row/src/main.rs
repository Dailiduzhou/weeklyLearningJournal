use std::cell::RefCell;
use std::collections::VecDeque;
use std::rc::Rc;

// Definition for a binary tree node.
#[derive(Debug, PartialEq, Eq)]
pub struct TreeNode {
    pub val: i32,
    pub left: Option<Rc<RefCell<TreeNode>>>,
    pub right: Option<Rc<RefCell<TreeNode>>>,
}

impl TreeNode {
    #[inline]
    pub fn new(val: i32) -> Self {
        TreeNode {
            val,
            left: None,
            right: None,
        }
    }
}

struct Solution;

impl Solution {
    pub fn largest_values(root: Option<Rc<RefCell<TreeNode>>>) -> Vec<i32> {
        let mut res = Vec::new();
        let mut que = VecDeque::new();
        if let Some(root) = root {
            que.push_back(root);
        }

        while !que.is_empty() {
            let level_size = que.len();
            let mut level_max = i32::MIN;

            for _ in 0..level_size {
                let node = que.pop_front().unwrap();
                let node = node.borrow();
                level_max = level_max.max(node.val);

                // Rc::clone 只增加引用计数，不复制节点，也不修改原树。
                if let Some(left) = &node.left {
                    que.push_back(Rc::clone(left));
                }
                if let Some(right) = &node.right {
                    que.push_back(Rc::clone(right));
                }
            }

            res.push(level_max);
        }

        res
    }
}

fn main() {
    let root = Some(Rc::new(RefCell::new(TreeNode::new(1))));
    println!("{:?}", Solution::largest_values(root));
}

#[cfg(test)]
mod tests {
    use super::*;

    fn node(
        val: i32,
        left: Option<Rc<RefCell<TreeNode>>>,
        right: Option<Rc<RefCell<TreeNode>>>,
    ) -> Option<Rc<RefCell<TreeNode>>> {
        Some(Rc::new(RefCell::new(TreeNode { val, left, right })))
    }

    #[test]
    fn empty_tree() {
        assert_eq!(Solution::largest_values(None), Vec::<i32>::new());
    }

    #[test]
    fn single_node() {
        assert_eq!(
            Solution::largest_values(node(i32::MIN, None, None)),
            vec![i32::MIN]
        );
    }

    #[test]
    fn multiple_levels() {
        // [1, 3, 2, 5, 3, null, 9]
        let root = node(
            1,
            node(3, node(5, None, None), node(3, None, None)),
            node(2, None, node(9, None, None)),
        );
        assert_eq!(Solution::largest_values(root.clone()), vec![1, 3, 9]);
        // 再次遍历仍得到相同结果，原树没有被破坏。
        assert_eq!(Solution::largest_values(root), vec![1, 3, 9]);
    }

    #[test]
    fn negative_values() {
        let root = node(-1, node(-5, None, None), node(-2, None, None));
        assert_eq!(Solution::largest_values(root), vec![-1, -2]);
    }

    #[test]
    fn skewed_tree() {
        let root = node(1, None, node(2, None, node(3, None, None)));
        assert_eq!(Solution::largest_values(root), vec![1, 2, 3]);
    }
}
