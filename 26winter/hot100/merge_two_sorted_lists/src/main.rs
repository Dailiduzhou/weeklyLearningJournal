struct Solution;

#[derive(PartialEq, Eq, Clone, Debug)]
pub struct ListNode {
    pub val: i32,
    pub next: Option<Box<ListNode>>,
}

impl ListNode {
    #[inline]
    fn new(val: i32) -> Self {
        ListNode { next: None, val }
    }
}
impl Solution {
    pub fn merge_two_lists(
        list1: Option<Box<ListNode>>,
        list2: Option<Box<ListNode>>,
    ) -> Option<Box<ListNode>> {
        let mut vec = Vec::new();

        let mut list1 = list1;
        let mut list2 = list2;
        while let Some(node) = list1 {
            vec.push(node.val);
            list1 = node.next;
        }

        while let Some(node) = list2 {
            vec.push(node.val);
            list2 = node.next;
        }

        vec.sort();
        let mut last_node = None;
        for &val in vec.iter().rev() {
            let mut node = ListNode::new(val);
            node.next = last_node;
            last_node = Some(Box::new(node));
        }

        last_node
    }
}

#[allow(unused)]
struct Solution1;

impl Solution1 {
    pub fn merge_two_lists(
        list1: Option<Box<ListNode>>,
        list2: Option<Box<ListNode>>,
    ) -> Option<Box<ListNode>> {
        let mut l1 = list1;
        let mut l2 = list2;

        // 1. 创建虚拟头节点 (Dummy Head)
        let mut dummy = Box::new(ListNode::new(0));

        // 2. 创建尾指针 (Tail Pointer)，初始指向 dummy
        let mut tail = &mut dummy;

        // 3. 当两个链表都不为空时循环
        while l1.is_some() && l2.is_some() {
            // 比较两个链表当前头节点的值
            // 注意：这里只是引用比较，没有移动所有权
            if l1.as_ref().unwrap().val < l2.as_ref().unwrap().val {
                // 如果 l1 较小，把 l1 接到 tail 后面
                tail.next = l1;
                // 更新 tail 指向刚接上的节点
                tail = tail.next.as_mut().unwrap();
                // 让 l1 向前移动（取出 tail 后面的 next，赋值给 l1）
                l1 = tail.next.take();
            } else {
                // 如果 l2 较小，同理
                tail.next = l2;
                tail = tail.next.as_mut().unwrap();
                l2 = tail.next.take();
            }
        }

        // 4. 处理剩余部分
        // 循环结束后，肯定有一个链表为空，另一个可能还有剩余节点
        // 直接把剩余的链表挂到 tail 后面即可
        tail.next = if l1.is_some() { l1 } else { l2 };

        // 5. 返回 dummy 的下一个节点（真正的头节点）
        dummy.next
    }
}

fn main() {
    println!("Hello world");
}

#[cfg(test)]
mod tests {
    use super::*;

    // 辅助函数：从数组创建链表
    fn create_list(vals: &[i32]) -> Option<Box<ListNode>> {
        if vals.is_empty() {
            return None;
        }

        let mut head = None;
        // 从后往前构建链表
        for &val in vals.iter().rev() {
            let mut node = ListNode::new(val);
            node.next = head;
            head = Some(Box::new(node));
        }
        head
    }

    // 辅助函数：比较两个链表是否相等
    fn list_equal(mut l1: Option<Box<ListNode>>, mut l2: Option<Box<ListNode>>) -> bool {
        while l1.is_some() && l2.is_some() {
            if l1.as_ref().unwrap().val != l2.as_ref().unwrap().val {
                return false;
            }
            l1 = l1.unwrap().next;
            l2 = l2.unwrap().next;
        }
        l1.is_none() && l2.is_none()
    }

    #[allow(dead_code)]
    fn print_list(mut head: Option<Box<ListNode>>) {
        let mut result = vec![];
        while let Some(node) = head {
            result.push(node.val.to_string());
            head = node.next;
        }
        println!("{}", result.join(" -> "));
    }

    #[test]
    fn test_both_empty() {
        let list1 = create_list(&[]);
        let list2 = create_list(&[]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[]);
        assert!(list_equal(result, expected), "两个空链表合并应该还是空链表");
    }

    #[test]
    fn test_first_empty() {
        let list1 = create_list(&[]);
        let list2 = create_list(&[1, 3, 5]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[1, 3, 5]);
        assert!(
            list_equal(result, expected),
            "第一个链表为空时，应返回第二个链表"
        );
    }

    #[test]
    fn test_second_empty() {
        let list1 = create_list(&[2, 4, 6]);
        let list2 = create_list(&[]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[2, 4, 6]);
        assert!(
            list_equal(result, expected),
            "第二个链表为空时，应返回第一个链表"
        );
    }

    #[test]
    fn test_both_non_empty_sorted() {
        let list1 = create_list(&[1, 3, 5]);
        let list2 = create_list(&[2, 4, 6]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[1, 2, 3, 4, 5, 6]);
        assert!(
            list_equal(result, expected),
            "两个有序链表应该合并成一个有序链表"
        );
    }

    #[test]
    fn test_duplicate_values() {
        let list1 = create_list(&[1, 2, 4]);
        let list2 = create_list(&[1, 3, 4]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[1, 1, 2, 3, 4, 4]);
        assert!(list_equal(result, expected), "有重复值的链表应该正确合并");
    }

    #[test]
    fn test_different_lengths() {
        let list1 = create_list(&[1, 2, 4, 7, 8]);
        let list2 = create_list(&[3, 5, 6]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[1, 2, 3, 4, 5, 6, 7, 8]);
        assert!(list_equal(result, expected), "长度不同的链表应该正确合并");
    }

    #[test]
    fn test_single_element_lists() {
        let list1 = create_list(&[5]);
        let list2 = create_list(&[3]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[3, 5]);
        assert!(list_equal(result, expected), "单元素链表应该正确合并");
    }

    #[test]
    fn test_descending_order() {
        // 注意：链表本身不一定是升序的，但通常测试用例都是有序链表
        // 这里测试其中一个链表是降序的情况
        let list1 = create_list(&[1, 3, 5]);
        let list2 = create_list(&[6, 4, 2]);
        let result = Solution::merge_two_lists(list1, list2);
        // 由于我们的实现使用排序，所以无论输入是否有序，输出都是升序的
        let expected = create_list(&[1, 2, 3, 4, 5, 6]);
        assert!(list_equal(result, expected), "无序链表合并后应该是有序的");
    }

    #[test]
    fn test_large_range() {
        let list1 = create_list(&[-100, 0, 100]);
        let list2 = create_list(&[-50, 50]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[-100, -50, 0, 50, 100]);
        assert!(
            list_equal(result, expected),
            "包含负数和正数的链表应该正确合并"
        );
    }

    #[test]
    fn test_all_same() {
        let list1 = create_list(&[5, 5, 5]);
        let list2 = create_list(&[5, 5, 5]);
        let result = Solution::merge_two_lists(list1, list2);
        let expected = create_list(&[5, 5, 5, 5, 5, 5]);
        assert!(
            list_equal(result, expected),
            "所有元素相同的链表应该正确合并"
        );
    }
}
