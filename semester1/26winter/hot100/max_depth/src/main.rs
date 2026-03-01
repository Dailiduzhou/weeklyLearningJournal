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
    pub fn max_depth(root: Option<Rc<RefCell<TreeNode>>>) -> i32 {
        fn helper(node: Option<&Rc<RefCell<TreeNode>>>) -> i32 {
            use std::cmp::max;

            match node {
                Some(node_ref) => {
                    let node = node_ref.borrow();

                    1 + max(helper(node.left.as_ref()), helper(node.right.as_ref()))
                }
                None => 0,
            }
        }
        helper(root.as_ref())
    }
}

struct Solution1;

impl Solution1 {
    pub fn max_depth(root: Option<Rc<RefCell<TreeNode>>>) -> i32 {
        match root {
            // 1. 基本情况：如果 Option 是 None，深度为 0
            None => 0,

            // 2. 如果是 Some(node)，我们需要拿到里面的内容
            Some(node) => {
                // 注意：这里的 node 类型是 Rc<RefCell<TreeNode>>

                // 3. borrow(): 获取 RefCell 内部数据的不可变引用
                // 我们需要查看 node.left 和 node.right
                let n = node.borrow();

                // 4. clone(): 因为递归函数 max_depth 需要所有权 (Option<Rc...>)
                // n.left 是 Option<Rc...>, 我们调用 clone() 只是复制了 Rc 指针 (引用计数+1)
                // 这是一个非常廉价的操作，并没有深拷贝整棵树
                let left_depth = Self::max_depth(n.left.clone());
                let right_depth = Self::max_depth(n.right.clone());

                1 + std::cmp::max(left_depth, right_depth)
            }
        }
    }
}
