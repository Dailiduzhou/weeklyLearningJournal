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
use std::collections::VecDeque;
use std::iter::once;
use std::rc::Rc;
struct Solution;

impl Solution {
    pub fn average_of_levels(root: Option<Rc<RefCell<TreeNode>>>) -> Vec<f64> {
        let mut q: VecDeque<_> = once(root).flatten().collect();
        let mut rez = vec![];
        while !q.is_empty() {
            let (mut sum, n) = (0.0, q.len());
            for _ in 0..n {
                let node_rc = q.pop_front().unwrap();
                let node_ref = node_rc.borrow();
                sum += node_ref.val as f64;
                q.extend(
                    once(node_ref.left.clone())
                        .chain(once(node_ref.right.clone()))
                        .flatten(),
                );
            }
            rez.push(sum / (n as f64));
        }
        rez
    }
}
