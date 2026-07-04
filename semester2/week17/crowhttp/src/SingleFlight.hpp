#pragma once
#include <functional>
#include <future>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>

template <typename T> class SingleFlight {
private:
  struct Call {
    std::promise<T> promise;
    std::shared_future<T> future;
    // 使用 shared_future 让多个线程可以多次 get()
    Call() : future(promise.get_future().share()) {}
  };

  std::mutex mu_;
  std::unordered_map<std::string, std::shared_ptr<Call>> calls_;

public:
  // 传入一个 Key 和一个返回值为 T 的 Lambda 表达式
  T execute(const std::string &key, std::function<T()> fn) {
    std::shared_ptr<Call> call;
    bool is_first = false;

    {
      std::lock_guard<std::mutex> lock(mu_);
      if (calls_.find(key) == calls_.end()) {
        call = std::make_shared<Call>();
        calls_[key] = call;
        is_first = true; // 我是第一个到达的线程
      } else {
        call = calls_[key]; // 发现已经有线程在执行了
      }
    }

    if (is_first) {
      try {
        // 执行真实耗时逻辑（如查数据库）
        T result = fn();
        call->promise.set_value(result);
      } catch (...) {
        call->promise.set_exception(std::current_exception());
      }

      // 执行完毕，从 map 中移除，防止内存泄漏
      std::lock_guard<std::mutex> lock(mu_);
      calls_.erase(key);
      return call->future.get();
    }

    // 非首个线程：直接阻塞等待 future 返回结果
    return call->future.get();
  }
};
