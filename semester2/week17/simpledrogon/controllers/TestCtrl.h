#pragma once

#include <drogon/HttpSimpleController.h>
#include <drogon/HttpTypes.h>

using namespace drogon;

class TestCtrl : public drogon::HttpSimpleController<TestCtrl> {
public:
  void asyncHandleHttpRequest(
      const HttpRequestPtr &req,
      std::function<void(const HttpResponsePtr &)> &&callback) override;
  PATH_LIST_BEGIN
  // PATH_ADD("/path", "filter1", "filter2", HttpMethod1, HttpMethod2...);
  PATH_ADD("/", Get, Post);
  PATH_ADD("/test", Get);
  PATH_LIST_END
};
