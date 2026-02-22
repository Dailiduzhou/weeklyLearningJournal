struct Solution;

impl Solution {
    pub fn subsets_with_dup(mut nums: Vec<i32>) -> Vec<Vec<i32>> {
        nums.sort_unstable();
        let n = nums.len();
        let mut result = Vec::new();

        for mask in 0..(1 << n) {
            let mut subset = Vec::new();
            let mut skip = false;

            for i in 0..n {
                if mask & (1 << i) != 0 {
                    // 如果当前位选中，且前一位相同但没选中，则跳过
                    if i > 0 && nums[i] == nums[i - 1] && mask & (1 << (i - 1)) == 0 {
                        skip = true;
                        break;
                    }
                    subset.push(nums[i]);
                }
            }

            if !skip {
                result.push(subset);
            }
        }

        result
    }
}

struct Solution1;

impl Solution1 {
    pub fn subsets_with_dup(mut nums: Vec<i32>) -> Vec<Vec<i32>> {
        // 1. 排序：让重复元素相邻，这是去重的前提
        nums.sort_unstable();
        let n = nums.len();
        // 2. 获取引用，方便在闭包中使用
        let nums = &nums;

        // 3. 生成所有可能的掩码 0 到 2^n - 1
        (0..1 << n)
            // 4. 过滤掉会产生重复子集的掩码
            .filter(|&mask| {
                (0..n).all(|i| {
                    // 如果第 i 位被选中
                    if (mask & (1 << i)) != 0 {
                        // 检查去重规则：
                        // 如果是重复元素 (nums[i] == nums[i-1])
                        // 且前一位没被选中 (mask & (1 << (i-1)) == 0)
                        // 则这个掩码非法
                        if i > 0 && nums[i] == nums[i - 1] && (mask & (1 << (i - 1))) == 0 {
                            return false;
                        }
                    }
                    true
                })
            })
            // 5. 将合法的掩码转换为实际的子集
            .map(|mask| {
                (0..n)
                    .filter(|&i| (mask & (1 << i)) != 0) // 只保留选中的位
                    .map(|i| nums[i]) // 取出对应的值
                    .collect() // 组成 Vec<i32>
            })
            .collect()
    }
}
