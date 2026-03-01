struct Solution;

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
use std::cell::RefCell;
use std::rc::Rc;
impl Solution {
    pub fn is_balanced(root: Option<Rc<RefCell<TreeNode>>>) -> bool {
        fn dfs(root: Option<Rc<RefCell<TreeNode>>>) -> Option<i32> {
            use std::cmp::max;
            match root {
                None => Some(0),
                Some(root) => {
                    let left_node = root.borrow().left.clone();
                    let right_node = root.borrow().right.clone();

                    let left = dfs(left_node)?;
                    let right = dfs(right_node)?;

                    if (left - right).abs() > 1 {
                        return None;
                    }

                    Some(1 + max(left, right))
                }
            }
        }

        dfs(root).is_some()
    }
}
