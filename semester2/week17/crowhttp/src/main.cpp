#include "SingleFlight.hpp"
#include "crow.h"
#include <iostream>
#include <pqxx/pqxx>
#include <sw/redis++/redis++.h>

extern "C" __attribute__((visibility("default"))) int yylex() { return 0; }
using namespace sw::redis;

namespace {
std::string env_or(const char *key, const char *fallback) {
  const char *v = std::getenv(key);
  return v && *v ? std::string(v) : std::string(fallback);
}
} // namespace

// 全局单例 SingleFlight，指定返回值为 std::string (通常是序列化后的 JSON
// 字符串)
SingleFlight<std::string> sf;

int main() {
  crow::SimpleApp app;

  // 初始化 Redis 连接 (REDIS_URL, 默认本地)
  auto redis = Redis(env_or("REDIS_URL", "tcp://127.0.0.1:6379"));

  // 配置 PostgreSQL 连接字符串 (PG_CONN_STR, 支持 libpq 连接串格式)
  std::string pg_conn_str = env_or(
      "PG_CONN_STR",
      "dbname=testdb user=postgres password=root host=127.0.0.1 port=5432");

  std::cout << "Redis: " << env_or("REDIS_URL", "tcp://127.0.0.1:6379") << "\n";
  std::cout << "PG:    " << pg_conn_str << "\n";

  CROW_ROUTE(app, "/user/<int>")
  ([&redis, pg_conn_str](int user_id) {
    std::string cache_key = "user:" + std::to_string(user_id);

    // 1. 尝试从 Redis 读取
    auto cached_val = redis.get(cache_key);
    if (cached_val) {
      // 缓存命中
      crow::response res(*cached_val);
      res.add_header("X-Cache", "HIT");
      return res;
    }

    // 2. 缓存未命中，使用 SingleFlight 防止并发击穿 DB
    try {
      std::string result = sf.execute(cache_key, [&]() -> std::string {
        std::cout << "Cache miss, hitting DB for user: " << user_id << "\n";

        // 3. 连接 PostgreSQL 查询数据
        pqxx::connection c(pg_conn_str);
        pqxx::work w(c);

        // 使用参数化查询防止 SQL 注入
        std::string sql = "SELECT name, age FROM users WHERE id = $1";
        pqxx::result db_res = w.exec_params(sql, user_id);

        if (db_res.empty()) {
          return "{}"; // 用户不存在
        }

        // 组装 JSON 结果
        crow::json::wvalue json_data;
        json_data["id"] = user_id;
        json_data["name"] = db_res[0]["name"].c_str();
        json_data["age"] = db_res[0]["age"].as<int>();

        std::string json_str = json_data.dump();

        // 4. 回写 Redis 缓存，设置 60 秒过期
        redis.setex(cache_key, 60, json_str);

        return json_str;
      });

      // 返回结果给 Crow
      crow::response res(result);
      res.add_header("X-Cache", "MISS");
      return res;

    } catch (const std::exception &e) {
      return crow::response(500, std::string("Internal Error: ") + e.what());
    }
  });

  app.port(18080).multithreaded().run();
  return 0;
}
