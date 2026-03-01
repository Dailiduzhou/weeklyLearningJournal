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
    pub fn delete_duplicates(mut head: Option<Box<ListNode>>) -> Option<Box<ListNode>> {
        let mut cur_opt = head.as_mut();

        while let Some(cur) = cur_opt {
            let mut next_opt = cur.next.take();

            while let Some(next) = next_opt.as_mut() {
                if cur.val == next.val {
                    next_opt = next.next.take();
                } else {
                    cur.next = next_opt;
                    break;
                }
            }
            cur_opt = cur.next.as_mut();
        }
        head
    }
}
