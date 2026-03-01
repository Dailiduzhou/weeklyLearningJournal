struct Solution;

impl Solution {
    pub fn subsets(nums: Vec<i32>) -> Vec<Vec<i32>> {
        let mut res = Vec::new();
        let mut subset = Vec::new();
        create_subset(&nums, 0, &mut res, &mut subset);
        res
    }
}

fn create_subset(nums: &Vec<i32>, i: usize, res: &mut Vec<Vec<i32>>, subset: &mut Vec<i32>) {
    if i == nums.len() {
        res.push(subset.clone());
        return;
    }

    subset.push(nums[i]);
    create_subset(nums, i + 1, res, subset);

    subset.pop();
    create_subset(nums, i + 1, res, subset);
}

fn main() {
    let nums = vec![1, 2, 3, 4];
    let res = Solution::subsets(nums);
    for i in res {
        for j in i {
            print!("{j} ");
        }
        println!();
    }
}
