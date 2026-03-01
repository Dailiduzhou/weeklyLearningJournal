#include <iostream>

struct ListNode {
  int val;
  ListNode *next;
  ListNode(int x) : val(x), next(NULL) {}
};

// Floyd判圈
class Solution {
public:
  ListNode *detectCycle(ListNode *head) {
    ListNode *fast = head;
    ListNode *slow = head;

    // 阶段 1: 判断是否有环
    while (fast != nullptr && fast->next != nullptr) {
      fast = fast->next->next;
      slow = slow->next;

      // 如果相遇，说明有环
      if (fast == slow) {
        // 阶段 2: 寻找环的入口
        // 将快指针重置回头部
        fast = head;

        // 两个指针同时每次走一步
        while (fast != slow) {
          fast = fast->next;
          slow = slow->next;
        }

        // 再次相遇点即为环入口
        return fast;
      }
    }

    // 退出循环说明无环
    return nullptr;
  }
};
