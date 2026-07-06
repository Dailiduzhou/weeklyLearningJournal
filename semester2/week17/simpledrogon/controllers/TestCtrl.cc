#include "TestCtrl.h"
#include <drogon/HttpResponse.h>
#include <drogon/HttpTypes.h>

void TestCtrl::asyncHandleHttpRequest(
    const HttpRequestPtr &req,
    std::function<void(const HttpResponsePtr &)> &&callback) {
  auto resp = HttpResponse::newHttpResponse();
  resp->setStatusCode(drogon::k200OK);
  resp->setContentTypeCode(drogon::CT_TEXT_HTML);
  resp->setBody("Hello World");
  callback(resp);
}
