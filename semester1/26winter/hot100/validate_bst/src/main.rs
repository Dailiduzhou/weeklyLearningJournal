use std::cell::RefCell;
use std::rc::Rc;

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
type Node = Rc<RefCell<TreeNode>>;
struct Solution;
impl Solution {
    pub fn is_valid_bst(root: Option<Node>) -> bool {
        // 空树视为有效 BST
        if root.is_none() {
            return true;
        }

        let mut last_val: Option<i32> = None;
        let mut ans = true;

        if let Some(root_node) = root {
            Solution::helper(root_node, &mut last_val, &mut ans);
        }

        ans
    }

    fn helper(root: Node, last_val: &mut Option<i32>, ans: &mut bool) {
        // 如果已经发现无效，提前终止遍历
        if !*ans {
            return;
        }

        let node = root.borrow();

        // 1. 遍历左子树
        if let Some(left) = node.left.clone() {
            Solution::helper(left, last_val, ans);
        }

        // 2. 处理当前节点（中序遍历的"根"部分）
        if let Some(prev_val) = *last_val {
            // 当前值必须严格大于上一个值
            if prev_val >= node.val {
                *ans = false;
                return;
            }
        }
        // 更新 last_val 为当前节点的值
        *last_val = Some(node.val);

        // 3. 遍历右子树
        if let Some(right) = node.right.clone() {
            Solution::helper(right, last_val, ans);
        }
    }
}

struct Solution1;
type OptNode = Option<Node>;

impl Solution1 {
    pub fn is_valid_bst(root: OptNode) -> bool {
        Self::is_valid(&root, i32::MIN as i64 - 1, i32::MAX as i64 + 1)
    }

    fn is_valid(node: &OptNode, gt: i64, lt: i64) -> bool {
        match node.as_ref() {
            None => true,
            Some(n) => {
                let b = n.borrow();
                (b.val as i64) > gt
                    && (b.val as i64) < lt
                    && Self::is_valid(&b.left, gt, b.val as i64)
                    && Self::is_valid(&b.right, b.val as i64, lt)
            }
        }
    }
}
