use std::env;
use std::process;
use minigrep::Config;  // 导入Config结构体

fn main() {    
   let config = Config::build(env::args()).unwrap_or_else(|err| {
        eprintln!("Problem parsing arguments: {err}");
        process::exit(1);
    });
    
    // 运行程序，处理错误
    if let Err(e) = minigrep::run(config) {
        eprintln!("程序执行错误: {e}");
        process::exit(1);
    }
}