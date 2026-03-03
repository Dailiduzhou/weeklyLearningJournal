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
        let mut q: VecDeque<Rc<RefCell<TreeNode>>> = once(root).flatten().collect();
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

struct Solution1;

impl Solution1 {
    pub fn average_of_levels(root: Option<Rc<RefCell<TreeNode>>>) -> Vec<f64> {
        let mut q: VecDeque<Rc<RefCell<TreeNode>>> = VecDeque::new();
        // 使用普通的 if let 来初始化队列，更直观
        if let Some(node) = root {
            q.push_back(node);
        }

        let mut rez = Vec::new();

        while !q.is_empty() {
            let n = q.len();
            let mut sum = 0.0;

            for _ in 0..n {
                // 这里我们确信队列不为空，直接 unwrap 是安全的
                let node_rc = q.pop_front().unwrap();
                let node_ref = node_rc.borrow();
                sum += node_ref.val as f64;

                // 用 if let 替代 once().chain().flatten()，避免迭代器开销
                if let Some(left) = &node_ref.left {
                    // 使用 Rc::clone(&left) 是 Rust 中克隆智能指针的推荐写法，以区别于深拷贝
                    q.push_back(Rc::clone(left));
                }
                if let Some(right) = &node_ref.right {
                    q.push_back(Rc::clone(right));
                }
            }
            rez.push(sum / (n as f64));
        }
        rez
    }
}
struct Solution2;

impl Solution2 {
    pub fn average_of_levels_dfs(root: Option<Rc<RefCell<TreeNode>>>) -> Vec<f64> {
        // 存储每一层的：(节点总和, 节点个数)
        let mut sums_counts: Vec<(f64, usize)> = Vec::new();

        // 辅助的递归函数
        fn dfs(
            node: &Option<Rc<RefCell<TreeNode>>>,
            level: usize,
            sums_counts: &mut Vec<(f64, usize)>,
        ) {
            if let Some(n) = node {
                let n = n.borrow();

                // 如果到达了一个全新的层级，初始化该层
                if level == sums_counts.len() {
                    sums_counts.push((0.0, 0));
                }

                // 累加当前层的值和节点数量
                sums_counts[level].0 += n.val as f64;
                sums_counts[level].1 += 1;

                // 继续遍历下一层
                dfs(&n.left, level + 1, sums_counts);
                dfs(&n.right, level + 1, sums_counts);
            }
        }

        // 从第 0 层开始递归
        dfs(&root, 0, &mut sums_counts);

        // 最后把 (总和, 个数) 映射成平均值返回
        sums_counts
            .into_iter()
            .map(|(sum, count)| sum / count as f64)
            .collect()
    }
}
