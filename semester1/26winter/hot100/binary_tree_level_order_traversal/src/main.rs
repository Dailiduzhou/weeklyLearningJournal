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
    pub fn level_order(root: Option<Rc<RefCell<TreeNode>>>) -> Vec<Vec<i32>> {
        use std::collections::VecDeque;
        let mut vd = VecDeque::new();
        if let Some(r) = root {
            vd.push_back(r);
        }

        let mut res = Vec::new();

        while !vd.is_empty() {
            let mut level = Vec::new();
            for _ in 0..vd.len() {
                if let Some(node) = vd.pop_front() {
                    level.push(node.borrow().val);
                    if let Some(l) = node.borrow_mut().left.take() {
                        vd.push_back(l);
                    }
                    if let Some(r) = node.borrow_mut().right.take() {
                        vd.push_back(r);
                    }
                }
            }
            res.push(level);
        }
        res
    }
}
