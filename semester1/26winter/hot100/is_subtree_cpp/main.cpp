
// Definition for a binary tree node.
struct TreeNode {
  int val;
  TreeNode *left;
  TreeNode *right;
  TreeNode() : val(0), left(nullptr), right(nullptr) {}
  TreeNode(int x) : val(x), left(nullptr), right(nullptr) {}
  TreeNode(int x, TreeNode *left, TreeNode *right)
      : val(x), left(left), right(right) {}
};

class Solution {
public:
  bool dev(TreeNode *r1, TreeNode *r2) {
    if (r1 == nullptr && r2 == nullptr)
      return true;
    if (r1 == nullptr || r2 == nullptr)
      return false;
    if (r1->val != r2->val)
      return false;
    return dev(r1->left, r2->left) && dev(r1->right, r2->right);
  }
  bool isSubtree(TreeNode *root, TreeNode *subRoot) {
    if (root == nullptr)
      return false;
    return dev(root, subRoot) || isSubtree(root->left, subRoot) ||
           isSubtree(root->right, subRoot);
  }
};
