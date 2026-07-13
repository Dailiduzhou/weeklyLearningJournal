#include <aws/core/Aws.h>
#include <drogon/drogon.h>

#include <cstdlib>
#include <string>
#include <stdexcept>

using namespace drogon;

namespace {

std::string envOr(const char *name, const std::string &fallback) {
  if (auto p = std::getenv(name)) return p;
  return fallback;
}

int envInt(const char *name, int fallback) {
  if (auto p = std::getenv(name)) {
    try { return std::stoi(p); } catch (...) {}
  }
  return fallback;
}

}  // namespace

int main() {
  Aws::SDKOptions options;
  Aws::InitAPI(options);

  const std::string host = envOr("HTTP_HOST", "0.0.0.0");
  const int port = envInt("HTTP_PORT", 8080);

  LOG_INFO << "Starting drogon-s3-demo on " << host << ":" << port;

  app().registerHandler(
      "/health",
      [](const HttpRequestPtr &,
         std::function<void(const HttpResponsePtr &)> &&cb) {
        auto r = HttpResponse::newHttpResponse();
        r->setBody("ok");
        cb(r);
      });

  // Configure AWS S3 (minIO / AWS) via env vars, used by S3Controller.
  //   S3_ENDPOINT            e.g. http://minio:9000  (empty => real AWS)
  //   AWS_REGION             e.g. us-east-1
  //   AWS_ACCESS_KEY_ID      e.g. minioadmin
  //   AWS_SECRET_ACCESS_KEY  e.g. minioadmin
  //   S3_BUCKET              e.g. test-bucket
  //   S3_PATH_STYLE          "1" (default) or "0"
  //   HTTP_HOST / HTTP_PORT  bind address
  setenv("AWS_REGION", envOr("AWS_REGION", "us-east-1").c_str(), 0);

  app().addListener(host, port).run();

  Aws::ShutdownAPI(options);
  return 0;
}
