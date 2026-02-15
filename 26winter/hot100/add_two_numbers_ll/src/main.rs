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

// dummy head
impl Solution {
    pub fn add_two_numbers(
        mut l1: Option<Box<ListNode>>,
        mut l2: Option<Box<ListNode>>,
    ) -> Option<Box<ListNode>> {
        let mut carry = 0;
        let mut head = Box::new(ListNode::new(0));
        let mut tail = &mut head;
        while l1.is_some() || l2.is_some() || carry != 0 {
            let val = match l1 {
                Some(n) => {
                    l1 = n.next;
                    n.val
                }
                None => 0,
            } + match l2 {
                Some(n) => {
                    l2 = n.next;
                    n.val
                }
                None => 0,
            } + carry;
            carry = val / 10;
            tail.next = Some(Box::new(ListNode::new(val % 10)));
            tail = tail.next.as_mut().unwrap();
        }
        head.next
    }
}

struct Solution1;

// recursive
impl Solution1 {
    pub fn add_two_numbers(
        l1: Option<Box<ListNode>>,
        l2: Option<Box<ListNode>>,
    ) -> Option<Box<ListNode>> {
        match (l1, l2) {
            (None, None) => None,
            (Some(n), None) | (None, Some(n)) => Some(n),
            (Some(n1), Some(n2)) => {
                let sum = n1.val + n2.val;
                if sum < 10 {
                    Some(Box::new(ListNode {
                        val: sum,
                        next: Solution::add_two_numbers(n1.next, n2.next),
                    }))
                } else {
                    let carry = Some(Box::new(ListNode::new(1)));
                    Some(Box::new(ListNode {
                        val: sum - 10,
                        next: Solution::add_two_numbers(
                            Solution::add_two_numbers(carry, n1.next),
                            n2.next,
                        ),
                    }))
                }
            }
        }
    }
}
