#pragma once

#include <drogon/HttpSimpleController.h>
#include <drogon/HttpTypes.h>

#include <aws/core/Aws.h>
#include <aws/s3/S3Client.h>

#include <memory>

using namespace drogon;

class S3Controller : public drogon::HttpSimpleController<S3Controller> {
public:
  void asyncHandleHttpRequest(
      const HttpRequestPtr &req,
      std::function<void(const HttpResponsePtr &)> &&callback) override;

  PATH_LIST_BEGIN
  PATH_ADD("/upload_test", drogon::Get);
  PATH_ADD("/s3-test", drogon::Get);
  PATH_LIST_END
};
