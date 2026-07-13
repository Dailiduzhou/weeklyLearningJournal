#include "S3Controller.h"

#include <aws/core/auth/AWSCredentialsProvider.h>
#include <aws/core/platform/Environment.h>
#include <aws/s3/S3ClientConfiguration.h>
#include <aws/s3/model/Bucket.h>
#include <aws/s3/model/CreateBucketRequest.h>
#include <aws/s3/model/PutObjectRequest.h>

#include <cstdlib>
#include <memory>
#include <sstream>
#include <string>

namespace {

struct S3Settings {
  std::string endpoint;     // empty => real AWS
  std::string region = "us-east-1";
  std::string accessKey;
  std::string secretKey;
  std::string bucket = "test-bucket";
  bool pathStyle = true;    // minIO requires path-style addressing
};

S3Settings loadSettings() {
  S3Settings s;
  if (auto p = std::getenv("S3_ENDPOINT")) s.endpoint = p;
  if (auto p = std::getenv("AWS_REGION")) s.region = p;
  if (auto p = std::getenv("AWS_ACCESS_KEY_ID")) s.accessKey = p;
  if (auto p = std::getenv("AWS_SECRET_ACCESS_KEY")) s.secretKey = p;
  if (auto p = std::getenv("S3_BUCKET")) s.bucket = p;
  if (auto p = std::getenv("S3_PATH_STYLE"))
    s.pathStyle = (std::string(p) != "0" && std::string(p) != "false");
  return s;
}

std::shared_ptr<Aws::S3::S3Client> makeClient(const S3Settings &s) {
  Aws::S3::S3ClientConfiguration cfg;
  cfg.region = s.region;
  cfg.useVirtualAddressing = !s.pathStyle;
  if (!s.endpoint.empty()) {
    cfg.endpointOverride = s.endpoint;
    cfg.scheme = s.endpoint.rfind("https://", 0) == 0 ? Aws::Http::Scheme::HTTPS
                                                       : Aws::Http::Scheme::HTTP;
  }
  if (!s.accessKey.empty() && !s.secretKey.empty()) {
    auto creds = Aws::MakeShared<Aws::Auth::SimpleAWSCredentialsProvider>(
        "S3Controller", s.accessKey, s.secretKey);
    return Aws::MakeShared<Aws::S3::S3Client>("S3Controller", creds, nullptr, cfg);
  }
  return Aws::MakeShared<Aws::S3::S3Client>("S3Controller", cfg);
}

bool ensureBucket(const Aws::S3::S3Client &client, const std::string &bucket) {
  Aws::S3::Model::CreateBucketRequest req;
  req.SetBucket(bucket.c_str());
  auto outcome = client.CreateBucket(req);
  if (outcome.IsSuccess()) return true;
  auto err = outcome.GetError();
  // "BucketAlready" => idempotent, treat as success
  if (err.GetErrorType() == Aws::S3::S3Errors::BUCKET_ALREADY_OWNED_BY_YOU ||
      err.GetErrorType() == Aws::S3::S3Errors::BUCKET_ALREADY_EXISTS ||
      std::string(err.GetMessage()).find("BucketAlready") != std::string::npos) {
    return true;
  }
  LOG_ERROR << "CreateBucket failed: " << err.GetMessage();
  return false;
}

}  // namespace

void S3Controller::asyncHandleHttpRequest(
    const HttpRequestPtr &req,
    std::function<void(const HttpResponsePtr &)> &&callback) {
  static const S3Settings settings = loadSettings();
  static const std::shared_ptr<Aws::S3::S3Client> client = makeClient(settings);
  static const bool bucketReady = ensureBucket(*client, settings.bucket);

  const std::string objectName = "test-file.txt";
  const std::string payload = "Hello from Drogon and AWS SDK for C++!";

  Aws::S3::Model::PutObjectRequest put;
  put.SetBucket(settings.bucket.c_str());
  put.SetKey(objectName.c_str());
  put.SetContentType("text/plain");

  auto body = Aws::MakeShared<Aws::StringStream>("S3Controller");
  *body << payload;
  put.SetBody(body);

  auto outcome = client->PutObject(put);

  auto resp = drogon::HttpResponse::newHttpResponse();
  std::ostringstream msg;
  if (outcome.IsSuccess()) {
    msg << "OK endpoint=" << (settings.endpoint.empty() ? "aws" : settings.endpoint)
        << " bucket=" << settings.bucket
        << " key=" << objectName
        << " bucketReady=" << (bucketReady ? "yes" : "no");
    resp->setBody(msg.str());
  } else {
    auto err = outcome.GetError();
    msg << "FAIL: " << err.GetExceptionName() << " - " << err.GetMessage();
    resp->setBody(msg.str());
    resp->setStatusCode(drogon::k500InternalServerError);
  }
  callback(resp);
}
