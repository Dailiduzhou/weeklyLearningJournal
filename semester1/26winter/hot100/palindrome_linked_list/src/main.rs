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
    pub fn is_palindrome(head: Option<Box<ListNode>>) -> bool {
        let mut list: Vec<i32> = Vec::new();
        let mut curr: Option<&Box<ListNode>> = head.as_ref();

        while let Some(node) = curr {
            list.push(node.val);
            curr = node.next.as_ref();
        }
        let (mut left, mut right) = (0, list.len().saturating_sub(1));
        while left < right {
            if list[left] != list[right] {
                return false;
            }
            left += 1;
            right -= 1;
        }
        true
    }
}
