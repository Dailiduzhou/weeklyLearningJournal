#[derive(PartialEq, Eq, Clone, Debug)]
pub struct ListNode {
    pub val: i32,
    pub next: Option<Box<ListNode>>,
}
//
impl ListNode {
    #[inline]
    fn new(val: i32) -> Self {
        ListNode { next: None, val }
    }
}

struct Solution;

impl Solution {
    pub fn remove_nth_from_end(head: Option<Box<ListNode>>, n: i32) -> Option<Box<ListNode>> {
        pub fn recurse(node: Option<Box<ListNode>>, n: i32) -> (Option<Box<ListNode>>, i32) {
            match node {
                None => (None, 0),
                Some(mut inner) => {
                    // 递归深入到链表尾部
                    let (next_node, idx_from_end) = recurse(inner.next.take(), n);

                    // 归（回溯）的时候计数
                    let current_idx = idx_from_end + 1;

                    if current_idx == n {
                        // 如果当前是倒数第 n 个，直接返回它的 next，相当于跳过了当前节点
                        (next_node, current_idx)
                    } else {
                        // 否则，把 next 接回来
                        inner.next = next_node;
                        (Some(inner), current_idx)
                    }
                }
            }
        }

        recurse(head, n).0
    }
    pub fn remove_nth_from_end_dummy(head: Option<Box<ListNode>>, n: i32) -> Option<Box<ListNode>> {
        // 1. 创建 Dummy Head 指向 head
        // 这样做是为了方便处理 "删除第一个节点" 这种特殊情况
        let mut dummy = Box::new(ListNode { val: 0, next: head });

        // 2. 计算链表长度
        let mut len = 0;
        {
            // 创建一个临时作用域进行只读遍历，
            // 这样 curr 的借用会在这个作用域结束时释放，
            // 不会影响后面我们对 dummy 的可变借用。
            let mut curr = dummy.next.as_ref();
            while let Some(node) = curr {
                len += 1;
                curr = node.next.as_ref();
            }
        } // curr 在这里“死”了，借用结束

        // 3. 计算需要移动的步数 (找到倒数第 N 个节点的前一个节点)
        let idx = len - n;

        // 4. 获取 Dummy 的可变引用，准备修改链表
        let mut curr = dummy.as_mut();

        // 走到要删除节点的前一个节点
        for _ in 0..idx {
            // 这里的 unwrap 是安全的，因为我们计算过长度，必然存在
            curr = curr.next.as_mut().unwrap();
        }

        // 5. 进行删除操作
        // 把 curr.next (要删除的节点) 替换为 curr.next.next
        let next_node = curr.next.as_mut().unwrap().next.take();
        curr.next = next_node;

        // 6. 返回 dummy.next (即新的 head)
        dummy.next
    }
    pub fn remove_nth_from_end_unsafe(
        head: Option<Box<ListNode>>,
        n: i32,
    ) -> Option<Box<ListNode>> {
        // 1. 创建 Dummy Head
        let mut dummy = Box::new(ListNode { val: 0, next: head });

        // 2. 获取 Dummy 的裸指针 (Raw Pointer)
        // &mut *dummy 的意思是：先把 Box 解引用拿到 ListNode，再取其可变引用
        // Rust 会自动把 &mut ListNode 强转为 *mut ListNode
        let dummy_ptr: *mut ListNode = &mut *dummy;

        unsafe {
            let mut fast: *mut ListNode = dummy_ptr;
            let mut slow: *mut ListNode = dummy_ptr;

            // 3. 让 fast 先走 n 步
            // 注意：我们需要走 n+1 步（因为从 dummy 开始），这样 slow 才会停在"被删节点"的前一个
            for _ in 0..=n {
                if fast.is_null() {
                    // 防御性代码，防止 n 超过链表长度
                    return dummy.next;
                }
                // 移动 fast 指针
                fast = Solution::step_forward(fast);
            }

            // 4. fast 和 slow 同时移动，直到 fast 走到链表末尾（即 null）
            while !fast.is_null() {
                fast = Solution::step_forward(fast);
                slow = Solution::step_forward(slow);
            }

            // 5. 进行删除操作
            // 此时 slow 指向待删除节点的前驱节点
            // (*slow).next 是 Option<Box<ListNode>>
            // take() 把 Box 取出来，链表中该位置变成 None
            let node_to_delete = (*slow).next.take();

            if let Some(node) = node_to_delete {
                // 把取出的节点的 next 接回 slow 后面
                (*slow).next = node.next;
            }
        }

        dummy.next
    }

    // 辅助函数：简化裸指针的移动逻辑
    // 作用：从 *mut ListNode 获取 next 的裸指针
    unsafe fn step_forward(ptr: *mut ListNode) -> *mut ListNode {
        // 1. (*ptr).next 获取 Option<Box<ListNode>>
        // 2. as_deref_mut() 变成 Option<&mut ListNode>
        // 3. map 将 &mut ListNode 转为 *mut ListNode
        // 4. unwrap_or 假如是 None 则返回空指针
        unsafe {
            (*ptr)
                .next
                .as_deref_mut()
                .map(|node| node as *mut ListNode)
                .unwrap_or(std::ptr::null_mut())
        }
    }
}
