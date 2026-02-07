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
    pub fn diameter_of_binary_tree(root: Option<Rc<RefCell<TreeNode>>>) -> i32 {
        // 定义递归函数 dfs
        // 返回值类型 (i32, i32):
        // tuple.0 (Depth):  从当前节点向下能延伸的最长路径长度（给父节点用的）
        // tuple.1 (Diameter): 当前子树内部找到的最大直径（最终答案的候选者）
        fn dfs(root: Option<Rc<RefCell<TreeNode>>>) -> (i32, i32) {
            match root {
                // Base Case: 空节点
                // 深度是 0，直径也是 0
                None => (0, 0),

                Some(node) => {
                    // borrow() 获取节点内容的引用
                    // clone() 增加 Rc 的引用计数（低开销），以便传入递归函数

                    // 1. 递归处理左子树
                    // ld = Left Depth, ldia = Left Diameter
                    let (ld, ldia) = dfs(node.borrow().left.clone());

                    // 2. 递归处理右子树
                    // rd = Right Depth, rdia = Right Diameter
                    let (rd, rdia) = dfs(node.borrow().right.clone());

                    // 3. 计算当前节点的深度 (Depth)
                    // 逻辑：我能提供的深度 = 左右子树中最深的那个 + 我自己(1)
                    // 这个值将作为 tuple.0 返回给上一层
                    let current_depth = i32::max(ld, rd) + 1;

                    // 4. 计算当前子树的最大直径 (Diameter)
                    // 真正的最大直径可能出现在三个地方：
                    // case A: 完全在左子树内部 (ldia)
                    // case B: 完全在右子树内部 (rdia)
                    // case C: 穿过当前节点，连接左右两边 (ld + rd)
                    // 取这三者中的最大值
                    let cross_root_path = ld + rd; // 穿过当前节点的路径长度
                    let max_sub_diameter = i32::max(ldia, rdia); // 子树内部已有的最大值
                    let current_diameter = i32::max(max_sub_diameter, cross_root_path);

                    // 返回 (深度, 直径)
                    (current_depth, current_diameter)
                }
            }
        }

        // 调用 dfs，只需要取元组的第二个元素（直径）作为最终结果
        dfs(root).1
    }
}
