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
    pub fn is_symmetric(root: Option<Rc<RefCell<TreeNode>>>) -> bool {
        if let Some(root) = root {
            let root = root.borrow();
            Solution::is_mirror_ref(&root.left, &root.right)
        } else {
            true
        }
    }

    fn is_mirror_ref(
        left: &Option<Rc<RefCell<TreeNode>>>,
        right: &Option<Rc<RefCell<TreeNode>>>,
    ) -> bool {
        match (left, right) {
            (Some(left), Some(right)) => {
                let left = left.borrow();
                let right = right.borrow();
                left.val == right.val
                    && Solution::is_mirror_ref(&left.left, &right.right)
                    && Solution::is_mirror_ref(&left.right, &right.left)
            }
            (None, None) => true,
            _ => false,
        }
    }
}
