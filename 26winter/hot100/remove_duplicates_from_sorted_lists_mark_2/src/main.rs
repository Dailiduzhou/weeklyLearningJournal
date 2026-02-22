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
    pub fn delete_duplicates(head: Option<Box<ListNode>>) -> Option<Box<ListNode>> {
        let mut dummy = Box::new(ListNode::new(0));
        dummy.next = head;
        let mut prev = &mut dummy;

        while let Some(mut current) = prev.next.take() {
            let duplicate_found =
                current.next.is_some() && current.val == current.next.as_ref().unwrap().val;

            if duplicate_found {
                let dup_val = current.val;

                while let Some(next) = current.next.take() {
                    if next.val == dup_val {
                        current = next;
                    } else {
                        current.next = Some(next);
                        break;
                    }
                }

                prev.next = current.next.take();
            } else {
                prev.next = Some(current);
                prev = prev.next.as_mut().unwrap();
            }
        }

        dummy.next
    }
}

struct Solution1;
impl Solution1 {
    pub fn delete_duplicates(head: Option<Box<ListNode>>) -> Option<Box<ListNode>> {
        let mut dummy = Box::new(ListNode { val: 0, next: head });

        let mut prev = &mut dummy;

        while let Some(mut curr) = prev.next.take() {
            let mut duplicated = false;
            // Skip all consecutive duplicates of `curr`
            while let Some(next) = &mut curr.next
                && curr.val == next.val
            {
                curr.next = next.next.take();
                duplicated = true;
            }

            if duplicated {
                // Skip `curr` itself
                prev.next = curr.next;
            } else {
                prev = prev.next.insert(curr);
            }
        }

        dummy.next
    }
}
