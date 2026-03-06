#include <cstddef>
#include <unordered_map>
class Node {
public:
  int val;
  Node *next;
  Node *random;

  Node(int _val) {
    val = _val;
    next = NULL;
    random = NULL;
  }
};

class Solution {
public:
  Node *copyRandomList(Node *head) {
    if (!head)
      return nullptr;

    std::pmr::unordered_map<Node *, Node *> old_2_new;

    Node *cur = head;
    while (cur) {
      old_2_new[cur] = new Node(cur->val);
      cur = cur->next;
    }

    cur = head;
    while (cur) {
      old_2_new[cur]->next = old_2_new[cur->next];
      old_2_new[cur]->random = old_2_new[cur->random];
      cur = cur->next;
    }
    return old_2_new[head];
  }
  Node *copyRandomList_O1(Node *head) {
    if (!head)
      return nullptr;

    Node *curr = head;
    while (curr) {
      Node *new_node = new Node(curr->val);
      new_node->next = curr->next; // 新节点指向原节点的下一个节点
      curr->next = new_node;       // 原节点指向新节点
      curr = new_node->next;       // 移动到下一个原节点
    }

    curr = head;
    while (curr) {
      if (curr->random) {
        // 新节点的 random = 原节点 random
        // 指向节点的下一个节点（即对应的复制节点）
        curr->next->random = curr->random->next;
      }
      curr = curr->next->next; // 每次跳跃两个节点，遍历原节点
    }
    Node *old_head = head;
    Node *new_head = head->next; // 最终要返回的新链表头
    Node *curr_old = old_head;
    Node *curr_new = new_head;

    while (curr_old) {
      // 恢复原链表：A 指向 B
      curr_old->next = curr_old->next->next;
      // 连接新链表：A' 指向 B'
      curr_new->next = curr_new->next ? curr_new->next->next : nullptr;

      // 指针往后移动
      curr_old = curr_old->next;
      curr_new = curr_new->next;
    }

    return new_head; // 返回深拷贝后的新链表
  }
};
