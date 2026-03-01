struct Solution;

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
use std::cell::RefCell;
use std::rc::Rc;
impl Solution {
    pub fn get_minimum_difference(root: Option<Rc<RefCell<TreeNode>>>) -> i32 {
        let mut data = vec![];
        Self::foo(root, &mut data);
        data.windows(2).map(|v| v[1] - v[0]).min().unwrap()
    }

    fn foo(root: Option<Rc<RefCell<TreeNode>>>, data: &mut Vec<i32>) {
        if let Some(root) = root {
            let mut root = root.borrow_mut();
            Self::foo(root.left.take(), data);
            data.push(root.val);
            Self::foo(root.right.take(), data);
        }
    }
}

struct Solution1;

impl Solution1 {
    pub fn get_minimum_difference(root: Option<Rc<RefCell<TreeNode>>>) -> i32 {
        let mut prev: Option<i32> = None;
        let mut min_diff = i32::MAX;

        Self::inorder(&root, &mut prev, &mut min_diff);

        min_diff
    }

    fn inorder(node: &Option<Rc<RefCell<TreeNode>>>, prev: &mut Option<i32>, min_diff: &mut i32) {
        if let Some(n) = node {
            let n = n.borrow();

            Self::inorder(&n.left, prev, min_diff);

            if let Some(val) = *prev {
                *min_diff = (*min_diff).min(n.val - val);
            }
            *prev = Some(n.val);

            Self::inorder(&n.right, prev, min_diff);
        }
    }
}
