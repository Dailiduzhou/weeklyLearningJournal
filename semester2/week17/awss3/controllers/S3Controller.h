#pragma once

#include <drogon/HttpSimpleController.h>
#include <drogon/HttpTypes.h>

using namespace drogon;

class S3Controller : public drogon::HttpSimpleController<S3Controller> {
public:
  void asyncHandleHttpRequest(
      const HttpRequestPtr &req,
      std::function<void(const HttpResponsePtr &)> &&callback) override;
  PATH_LIST_BEGIN
  // list path definitions here;
  // PATH_ADD("/path", "filter1", "filter2", HttpMethod1, HttpMethod2...);
  PATH_ADD("/upload_test", drogon::Get);
  PATH_LIST_END
};
