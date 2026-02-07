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

struct Solution;

impl Solution {
    pub fn reverse_list(head: Option<Box<ListNode>>) -> Option<Box<ListNode>> {
        let (mut prev, mut curr) = (None, head);
        while let Some(mut node) = curr {
            curr = node.next;

            node.next = prev;

            prev = Some(node);
        }
        prev
    }

    pub fn reverse_lists(head: Option<Box<ListNode>>) -> Option<Box<ListNode>> {
        std::iter::successors(head.as_ref(), |n| n.next.as_ref())
            .map(|n| n.val)
            .fold(None, |p, n| Some(Box::new(ListNode { val: n, next: p })))
    }
}
