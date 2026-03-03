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
use std::cell::RefCell;
use std::rc::Rc;
impl Solution {
    pub fn is_subtree(
        root: Option<Rc<RefCell<TreeNode>>>,
        sub_root: Option<Rc<RefCell<TreeNode>>>,
    ) -> bool {
        if root == sub_root {
            return true;
        }

        if let Some(node) = root {
            let node = node.borrow();

            Self::is_subtree(node.left.clone(), sub_root.clone())
                || Self::is_subtree(node.right.clone(), sub_root.clone())
        } else {
            false
        }
    }
}

struct Solution1;
impl Solution1 {
    pub fn is_subtree(
        root: Option<Rc<RefCell<TreeNode>>>,
        sub_root: Option<Rc<RefCell<TreeNode>>>,
    ) -> bool {
        use std::collections::VecDeque;

        let sub_root: Rc<RefCell<TreeNode>> = sub_root.unwrap();
        let mut queue: VecDeque<Rc<RefCell<TreeNode>>> = VecDeque::new();
        queue.push_back(root.unwrap());

        while let Some(node) = queue.pop_front() {
            if node == sub_root {
                return true;
            }

            if let Some(left) = node.borrow().left.clone() {
                queue.push_back(left);
            }
            if let Some(right) = node.borrow().right.clone() {
                queue.push_back(right);
            }
        }

        false
    }
}
