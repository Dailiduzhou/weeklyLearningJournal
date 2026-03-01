struct Solution;

impl Solution {
    pub fn max_profit(prices: Vec<i32>) -> i32 {
        let mut buy = prices[0];
        let mut profit = 0;
        for i in prices {
            if i < buy {
                buy = i;
            } else if i - buy > profit {
                profit = i - buy;
            }
        }
        profit
    }
}

fn main() {
    let prices = vec![7, 1, 5, 3, 6, 4];
    let res = Solution::max_profit(prices);
    println!("{res}");

    let prices = vec![7, 6, 4, 3, 1];
    let res = Solution::max_profit(prices);
    println!("{res}");
}
