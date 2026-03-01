// Definition for singly-linked list.
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
    pub fn middle_node(head: Option<Box<ListNode>>) -> Option<Box<ListNode>> {
        let mut fast = &head;
        let mut slow = &head;

        while fast.is_some() && fast.as_ref().unwrap().next.is_some() {
            slow = &slow.as_ref().unwrap().next;
            fast = &fast.as_ref().unwrap().next.as_ref().unwrap().next;
        }

        slow.clone()
    }

    pub fn middle_node1(head: Option<Box<ListNode>>) -> Option<Box<ListNode>> {
        let mut fast = head.clone().and_then(|x| x.next);
        let mut slow = head;

        while let Some(t) = fast {
            fast = t.next.and_then(|x| x.next);
            slow = slow.and_then(|x| x.next);
        }

        slow
    }
}
