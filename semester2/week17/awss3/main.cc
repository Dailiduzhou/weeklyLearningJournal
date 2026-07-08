#include <aws/core/Aws.h>
#include <aws/s3/S3Client.h>
#include <drogon/drogon.h>

using namespace drogon;

int main() {
  // 1. 初始化 AWS SDK
  Aws::SDKOptions options;
  // 如果需要可以配置日志等选项：options.loggingOptions.logLevel =
  // Aws::Utils::Logging::LogLevel::Info;
  Aws::InitAPI(options);

  // 2. 注册一个简单的 Drogon 路由来测试 S3 Client
  app().registerHandler(
      "/s3-test", [](const HttpRequestPtr &req,
                     std::function<void(const HttpResponsePtr &)> &&callback) {
        // 实例化 S3 Client
        Aws::Client::ClientConfiguration clientConfig;
        clientConfig.region = "us-east-1"; // 设置你的目标区域

        Aws::S3::S3Client s3_client(clientConfig);

        auto resp = HttpResponse::newHttpResponse();
        resp->setBody("AWS S3 Client Initialized in Drogon!");
        callback(resp);
      });

  // 3. 启动 Drogon HTTP 服务器 (这会阻塞当前线程直到服务器关闭)
  LOG_INFO << "Server running on 127.0.0.1:8080";
  app().addListener("127.0.0.1", 8080).run();

  // 4. Drogon 关闭后，清理 AWS SDK 资源
  Aws::ShutdownAPI(options);

  return 0;
}
