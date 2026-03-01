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

use std::cell::RefCell;
use std::rc::Rc;
impl Solution {
    pub fn has_path_sum(root: Option<Rc<RefCell<TreeNode>>>, target_sum: i32) -> bool {
        Solution::helper(root, target_sum, 0)
    }

    fn helper(root: Option<Rc<RefCell<TreeNode>>>, target_sum: i32, cur_sum: i32) -> bool {
        if let Some(node) = root {
            let sum = cur_sum + node.borrow().val;
            let left = node.borrow().left.clone();
            let right = node.borrow().right.clone();

            match (left, right) {
                (None, None) => target_sum == sum,
                (l, r) => {
                    Solution::helper(l, target_sum, sum) || Solution::helper(r, target_sum, sum)
                }
            }
        } else {
            false
        }
    }
}
