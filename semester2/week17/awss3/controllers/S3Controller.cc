#include "S3Controller.h"

void S3Controller::asyncHandleHttpRequest(
    const HttpRequestPtr &req,
    std::function<void(const HttpResponsePtr &)> &&callback) {
  // 1. 配置 S3 客户端
  Aws::Client::ClientConfiguration clientConfig;
  clientConfig.region = "us-east-1"; // 替换为你的 S3 存储桶区域
  Aws::S3::S3Client s3Client(clientConfig);

  const Aws::String bucketName = "your-bucket-name";
  const Aws::String objectName = "test-file.txt";

  // 2. 构造上传请求
  Aws::S3::Model::PutObjectRequest putObjectRequest;
  putObjectRequest.SetBucket(bucketName);
  putObjectRequest.SetKey(objectName);

  // 设置要上传的内存数据
  auto inputData = Aws::MakeShared("PutObjectInputStream");
  *inputData << "Hello from Drogon and AWS SDK for C++!";
  putObjectRequest.SetBody(inputData);

  // 3. 执行上传 (同步调用)
  auto putObjectOutcome = s3Client.PutObject(putObjectRequest);

  // 4. 返回 HTTP 响应给客户端
  auto resp = drogon::HttpResponse::newHttpResponse();
  if (putObjectOutcome.IsSuccess()) {
    resp->setBody("Successfully uploaded to S3!");
  } else {
    resp->setBody("S3 Upload Failed: " +
                  putObjectOutcome.GetError().GetMessage());
    resp->setStatusCode(drogon::k500InternalServerError);
  }

  callback(resp);
}
