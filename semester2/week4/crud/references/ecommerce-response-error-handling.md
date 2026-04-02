# E-commerce System Response & Error Handling Best Practices

## Overview

This guide covers comprehensive best practices for implementing unified response structures and error handling in go-zero e-commerce systems, including business error codes, HTTP status mapping, and real-world examples.

---

## Table of Contents

1. [Why Unified Response Structure?](#why-unified-response-structure)
2. [Framework Default Error Response Analysis](#framework-default-error-response-analysis)
3. [Best Practice Solution](#best-practice-solution)
4. [Implementation Details](#implementation-details)
5. [Real Response Examples](#real-response-examples)
6. [E-commerce Complete Example](#e-commerce-complete-example)
7. [Advanced Features](#advanced-features)
8. [Summary & Decision Checklist](#summary--decision-checklist)

---

## Why Unified Response Structure?

### Frontend Requirements

E-commerce frontend applications need:
- **Unified error code system** for i18n and user-friendly error messages
- **Consistent response format** for easier frontend processing
- **Clear business status distinction** between success and various error types

### Recommended Unified Response Structure

```go
// Unified response structure
type Response struct {
    Code    int         `json:"code"`    // Business status code
    Message string      `json:"message"` // User-friendly message
    Data    interface{} `json:"data"`    // Business data payload
}
```

### Business Code Design

```go
// Business code conventions
const (
    // 0: Success
    CodeSuccess = 0
    
    // 1-9999: Common errors
    CodeInvalidParam    = 1001  // Invalid parameters
    CodeUnauthorized    = 1002  // Unauthorized
    CodeForbidden       = 1003  // Permission denied
    CodeNotFound        = 1004  // Resource not found
    CodeInternalError   = 1005  // Internal server error
    
    // 10000-19999: User module
    CodeUserNotFound     = 10001  // User not found
    CodeUserDisabled     = 10002  // User account disabled
    CodePasswordWrong    = 10003  // Wrong password
    CodeUserExists       = 10004  // User already exists
    
    // 20000-29999: Product module
    CodeProductNotFound  = 20001  // Product not found
    CodeProductOffShelf  = 20002  // Product off shelf
    CodeStockNotEnough   = 20003  // Insufficient stock
    
    // 30000-39999: Order module
    CodeOrderNotFound    = 30001  // Order not found
    CodeOrderCanceled    = 30002  // Order canceled
    CodeOrderPaid        = 30003  // Order already paid
    CodeOrderExpired     = 30004  // Order expired
    
    // 40000-49999: Payment module
    CodePayFailed        = 40001  // Payment failed
    CodeRefundFailed     = 40002  // Refund failed
    CodeBalanceNotEnough = 40003  // Insufficient balance
)
```

### Response Examples

**Success Response:**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "orderId": "202401010001",
        "totalAmount": 199.99
    }
}
```

**Error Response:**
```json
{
    "code": 20003,
    "message": "Insufficient stock",
    "data": null
}
```

---

## Framework Default Error Response Analysis

### Current Project Problem

Your current implementation:

```go
// Logic layer
return nil, errorx.Wrapf(nil, "user not found")

// Handler layer
httpx.ErrorCtx(r.Context(), w, err)
```

**Framework default behavior (without custom error handler):**

```
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8

user not found
```

### Problems

- ❌ **Wrong HTTP status code** (should be 404, not 400)
- ❌ **Inconsistent response format** (plain text, not JSON)
- ❌ **No business error code**
- ❌ **Frontend cannot distinguish different error types**

---

## Best Practice Solution

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                   HTTP Request                           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  Handler Layer                                          │
│  - Parse HTTP request                                   │
│  - Call Logic layer                                     │
│  - Auto handle response (OkHandler/ErrorHandler)        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  Logic Layer                                            │
│  - Business logic                                       │
│  - Return business errors (BizError)                    │
│  - Return business data                                 │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  Response Handler (Global handlers)                     │
│                                                          │
│  Success: httpx.SetOkHandler                            │
│    - Wrap to: {code:0, message:"success", data}         │
│                                                          │
│  Error: httpx.SetErrorHandler                           │
│    - Map HTTP status codes                              │
│    - Return business error code and message             │
└─────────────────────────────────────────────────────────┘
```

### Solution Benefits

- ✅ **Clean three-layer architecture**
- ✅ **Unified response format**
- ✅ **Business error codes for i18n**
- ✅ **Proper HTTP status codes**
- ✅ **Frontend-friendly**

---

## Implementation Details

### Step 1: Define Business Error Type

```go
// internal/errors/errors.go
package errors

import (
    "errors"
    "net/http"
)

// BizError represents a business error with HTTP code and business code
type BizError struct {
    HTTPCode int         // HTTP status code
    BizCode  int         // Business error code
    Message  string      // Error message
    Details  interface{} // Optional error details
}

func (e *BizError) Error() string {
    return e.Message
}

// NewBizError creates a new business error
func NewBizError(httpCode, bizCode int, message string) *BizError {
    return &BizError{
        HTTPCode: httpCode,
        BizCode:  bizCode,
        Message:  message,
    }
}

// Predefined business errors
var (
    // User related
    ErrUserNotFound   = NewBizError(http.StatusNotFound, 10001, "User not found")
    ErrUserDisabled   = NewBizError(http.StatusForbidden, 10002, "User account disabled")
    ErrPasswordWrong  = NewBizError(http.StatusUnauthorized, 10003, "Invalid username or password")
    ErrUserExists     = NewBizError(http.StatusConflict, 10004, "Username already exists")
    
    // Product related
    ErrProductNotFound = NewBizError(http.StatusNotFound, 20001, "Product not found")
    ErrProductOffShelf = NewBizError(http.StatusForbidden, 20002, "Product is off shelf")
    ErrStockNotEnough  = NewBizError(http.StatusBadRequest, 20003, "Insufficient stock")
    
    // Order related
    ErrOrderNotFound  = NewBizError(http.StatusNotFound, 30001, "Order not found")
    ErrOrderCanceled  = NewBizError(http.StatusBadRequest, 30002, "Order has been canceled")
    ErrOrderPaid      = NewBizError(http.StatusBadRequest, 30003, "Order already paid")
    
    // Payment related
    ErrPayFailed        = NewBizError(http.StatusBadRequest, 40001, "Payment failed")
    ErrBalanceNotEnough = NewBizError(http.StatusBadRequest, 40003, "Insufficient balance")
)
```

### Step 2: Define Unified Response Structure

```go
// internal/types/response.go
package types

// Response unified response structure
type Response struct {
    Code    int         `json:"code"`    // Business status code (0=success)
    Message string      `json:"message"` // User-friendly message
    Data    interface{} `json:"data"`    // Business data payload
}

// Success creates a success response
func Success(data interface{}) *Response {
    return &Response{
        Code:    0,
        Message: "success",
        Data:    data,
    }
}

// Error creates an error response
func Error(code int, message string) *Response {
    return &Response{
        Code:    code,
        Message: message,
        Data:    nil,
    }
}
```

### Step 3: Register Global Handlers

```go
// cmd/user.go (or main entry point)
package main

import (
    "net/http"
    
    "crud/cmd/internal/config"
    "crud/cmd/internal/errors"
    "crud/cmd/internal/handler"
    "crud/cmd/internal/svc"
    "crud/cmd/internal/types"
    
    "github.com/zeromicro/go-zero/core/conf"
    "github.com/zeromicro/go-zero/rest"
    "github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

func main() {
    flag.Parse()
    
    var c config.Config
    conf.MustLoad(*configFile, &c)
    
    // 1. Register success response handler
    httpx.SetOkHandler(func(ctx context.Context, data interface{}) interface{} {
        // If already Response type, return as-is
        if resp, ok := data.(*types.Response); ok {
            return resp
        }
        // Otherwise wrap in unified format
        return types.Success(data)
    })
    
    // 2. Register error response handler
    httpx.SetErrorHandler(func(err error) (int, interface{}) {
        // Handle business errors
        if bizErr, ok := err.(*errors.BizError); ok {
            return bizErr.HTTPCode, &types.Response{
                Code:    bizErr.BizCode,
                Message: bizErr.Message,
                Data:    bizErr.Details,
            }
        }
        
        // Handle other errors
        return http.StatusInternalServerError, &types.Response{
            Code:    500,
            Message: "Internal server error",
            Data:    nil,
        }
    })
    
    server := rest.MustNewServer(c.RestConf)
    defer server.Stop()
    
    ctx := svc.NewServiceContext(c)
    handler.RegisterHandlers(server, ctx)
    
    server.Start()
}
```

### Step 4: Use in Logic Layer

```go
// internal/logic/user/loginlogic.go
func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResp, err error) {
    // Query user
    user, err := l.svcCtx.UserModel.FindOneByUsername(l.ctx, req.Username)
    if err != nil {
        if err == model.ErrNotFound {
            return nil, errors.ErrPasswordWrong  // Return business error
        }
        return nil, err
    }
    
    // Check user status
    if user.Status == "disabled" {
        return nil, errors.ErrUserDisabled  // Return business error
    }
    
    // Verify password
    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
    if err != nil {
        return nil, errors.ErrPasswordWrong  // Return business error
    }
    
    // Return business data (will be auto-wrapped by OkHandler)
    return &types.LoginResp{
        Username:     user.Username,
        AccessToken:  "xxx",
        RefreshToken: "yyy",
    }, nil
}

// internal/logic/order/createorderlogic.go
func (l *CreateOrderLogic) CreateOrder(req *types.CreateOrderRequest) (resp *types.CreateOrderResp, err error) {
    // Check product
    product, err := l.svcCtx.ProductModel.FindOne(l.ctx, req.ProductId)
    if err != nil {
        if err == model.ErrNotFound {
            return nil, errors.ErrProductNotFound
        }
        return nil, err
    }
    
    // Check stock
    if product.Stock < req.Quantity {
        return nil, errors.ErrStockNotEnough  // Return business error
    }
    
    // Check product status
    if product.Status != "on_shelf" {
        return nil, errors.ErrProductOffShelf
    }
    
    // Create order...
    return &types.CreateOrderResp{
        OrderId:     "202401010001",
        TotalAmount: product.Price * float64(req.Quantity),
    }, nil
}
```

---

## Real Response Examples

### Success Response

**Request:**
```http
POST /api/v1/user/login
Content-Type: application/json

{
  "username": "alice",
  "password": "password123"
}
```

**Response:**
```json
HTTP/1.1 200 OK
Content-Type: application/json

{
    "code": 0,
    "message": "success",
    "data": {
        "username": "alice",
        "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
}
```

### Business Error Responses

**Case 1: User Not Found**
```json
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
    "code": 10003,
    "message": "Invalid username or password",
    "data": null
}
```

**Case 2: Insufficient Stock**
```json
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
    "code": 20003,
    "message": "Insufficient stock",
    "data": null
}
```

**Case 3: Order Not Found**
```json
HTTP/1.1 404 Not Found
Content-Type: application/json

{
    "code": 30001,
    "message": "Order not found",
    "data": null
}
```

---

## E-commerce Complete Example

### Order Creation Flow

```go
// internal/logic/order/createorderlogic.go
func (l *CreateOrderLogic) CreateOrder(req *types.CreateOrderRequest) (*types.CreateOrderResp, error) {
    // 1. Validate product
    product, err := l.svcCtx.ProductModel.FindOne(l.ctx, req.ProductId)
    if err != nil {
        if err == model.ErrNotFound {
            return nil, errors.ErrProductNotFound
        }
        return nil, err
    }
    
    // 2. Validate stock
    if product.Stock < req.Quantity {
        return nil, &errors.BizError{
            HTTPCode: http.StatusBadRequest,
            BizCode:  errors.CodeStockNotEnough,
            Message:  fmt.Sprintf("Insufficient stock, current stock: %d", product.Stock),
            Details: map[string]interface{}{
                "current_stock": product.Stock,
                "requested":     req.Quantity,
            },
        }
    }
    
    // 3. Validate user balance (if paying with balance)
    if req.PaymentMethod == "balance" {
        user, _ := l.svcCtx.UserModel.FindOne(l.ctx, req.UserId)
        totalAmount := product.Price * float64(req.Quantity)
        if user.Balance < totalAmount {
            return nil, &errors.BizError{
                HTTPCode: http.StatusBadRequest,
                BizCode:  errors.CodeBalanceNotEnough,
                Message:  "Insufficient balance",
                Details: map[string]interface{}{
                    "required":   totalAmount,
                    "balance":    user.Balance,
                    "difference": totalAmount - user.Balance,
                },
            }
        }
    }
    
    // 4. Create order (in transaction)
    order, err := l.createOrderInTransaction(product, req)
    if err != nil {
        return nil, err
    }
    
    // 5. Return success response
    return &types.CreateOrderResp{
        OrderId:     order.OrderId,
        TotalAmount: order.TotalAmount,
        PaymentUrl:  order.PaymentUrl,
    }, nil
}
```

### Response Examples

**Success:**
```json
HTTP/1.1 200 OK

{
    "code": 0,
    "message": "success",
    "data": {
        "order_id": "202401010001",
        "total_amount": 199.99,
        "payment_url": "https://pay.example.com/order/202401010001"
    }
}
```

**Insufficient Stock:**
```json
HTTP/1.1 400 Bad Request

{
    "code": 20003,
    "message": "Insufficient stock, current stock: 5",
    "data": {
        "current_stock": 5,
        "requested": 10
    }
}
```

**Insufficient Balance:**
```json
HTTP/1.1 400 Bad Request

{
    "code": 40003,
    "message": "Insufficient balance",
    "data": {
        "required": 199.99,
        "balance": 50.00,
        "difference": 149.99
    }
}
```

---

## Advanced Features

### 1. Error Details with Field Information

```go
// Error with detailed information (e.g., which fields failed validation)
type BizError struct {
    HTTPCode int
    BizCode  int
    Message  string
    Details  interface{} `json:"details,omitempty"` // Error details
}

// Usage example
return nil, &errors.BizError{
    HTTPCode: http.StatusBadRequest,
    BizCode:  1001,
    Message:  "Validation failed",
    Details: map[string]string{
        "field": "password",
        "rule":  "min=8",
    },
}

// Response
{
    "code": 1001,
    "message": "Validation failed",
    "details": {
        "field": "password",
        "rule": "min=8"
    }
}
```

### 2. Internationalization (i18n)

```go
// Return different messages based on request language
httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
    lang := ctx.Value("lang").(string) // Get language from context
    
    var message string
    if bizErr, ok := err.(*errors.BizError); ok {
        // Select message based on language
        if lang == "zh" {
            message = bizErr.Message
        } else {
            message = bizErr.MessageEn // English message
        }
        
        return bizErr.HTTPCode, &types.Response{
            Code:    bizErr.BizCode,
            Message: message,
        }
    }
    
    return http.StatusInternalServerError, &types.Response{
        Code:    500,
        Message: "Internal Server Error",
    }
})
```

### 3. Error Logging

```go
httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
    // Log error
    logx.WithContext(ctx).Errorf("API error: %v", err)
    
    // Handle error...
    if bizErr, ok := err.(*errors.BizError); ok {
        return bizErr.HTTPCode, &types.Response{
            Code:    bizErr.BizCode,
            Message: bizErr.Message,
        }
    }
    
    return http.StatusInternalServerError, &types.Response{
        Code:    500,
        Message: "Internal server error",
    }
})
```

### 4. Request Tracing

```go
httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
    // Get trace ID from context
    traceID := logx.GetTraceIdFromContext(ctx)
    
    if bizErr, ok := err.(*errors.BizError); ok {
        return bizErr.HTTPCode, &types.Response{
            Code:    bizErr.BizCode,
            Message: bizErr.Message,
            TraceId: traceID, // Include trace ID in response
        }
    }
    
    return http.StatusInternalServerError, &types.Response{
        Code:    500,
        Message: "Internal server error",
        TraceId: traceID,
    }
})
```

### 5. Error Monitoring & Alerting

```go
httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
    // Send critical errors to monitoring system
    if bizErr, ok := err.(*errors.BizError); ok {
        // Track business error metrics
        metrics.Counter("api_error", map[string]string{
            "code":  string(bizErr.BizCode),
            "error": bizErr.Message,
        }).Inc()
        
        return bizErr.HTTPCode, &types.Response{
            Code:    bizErr.BizCode,
            Message: bizErr.Message,
        }
    }
    
    // Track unexpected errors
    metrics.Counter("api_error", map[string]string{
        "code":  "500",
        "error": "unexpected",
    }).Inc()
    
    return http.StatusInternalServerError, &types.Response{
        Code:    500,
        Message: "Internal server error",
    }
})
```

---

## Summary & Decision Checklist

### Best Practices Summary

| Practice | Solution | Required |
|----------|----------|----------|
| **Unified response format** | `SetOkHandler` + `SetErrorHandler` | ✅ Yes |
| **Business error codes** | Custom `BizError` type | ✅ Yes |
| **HTTP status code mapping** | Define in `BizError` | ✅ Yes |
| **Global handlers** | Register in `main()` | ✅ Yes |
| **Logic returns business errors** | Return `BizError` instances | ✅ Yes |
| **Error details** | `BizError.Details` field | ⚠️ Optional |
| **Internationalization** | Handle in `ErrorHandler` | ⚠️ Optional |
| **Error logging** | Use `ErrorHandlerCtx` | ⚠️ Optional |
| **Request tracing** | Include trace ID in response | ⚠️ Optional |
| **Error monitoring** | Integrate with metrics system | ⚠️ Optional |

### Do's and Don'ts

#### ✅ Do

1. **Use unified response format** for all API endpoints
2. **Define business error codes** in a centralized location
3. **Return domain errors** from Logic layer, not HTTP codes
4. **Use global handlers** to ensure consistency
5. **Keep handlers thin** - only HTTP parsing and routing
6. **Map HTTP status codes** appropriately (404, 401, 403, 400, 500)
7. **Include error details** for validation failures
8. **Log errors** for debugging and monitoring

#### ❌ Don't

1. **Don't return plain text errors** - always use JSON
2. **Don't mix HTTP status codes** in Logic layer
3. **Don't hardcode error messages** - use predefined constants
4. **Don't skip error handling** - always handle errors explicitly
5. **Don't expose internal errors** to clients
6. **Don't use different response formats** in the same project
7. **Don't forget to log errors** for production systems

### Architecture Principles

```
┌─────────────────────────────────────────────────────────┐
│  Logic Layer (Business Logic)                           │
│                                                          │
│  ✅ Return domain errors: ErrUserNotFound               │
│  ✅ Return business data: &LoginResp{...}               │
│  ❌ Don't set HTTP codes: http.StatusNotFound           │
│  ❌ Don't return Response: &Response{code:0, ...}       │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│  Handler Layer (HTTP Layer)                             │
│                                                          │
│  ✅ Parse HTTP request                                  │
│  ✅ Call Logic layer                                    │
│  ✅ Use httpx.OkJsonCtx / ErrorCtx                      │
│  ❌ Don't contain business logic                        │
│  ❌ Don't handle error mapping (use global handler)     │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│  Global Handlers (Response Formatting)                  │
│                                                          │
│  ✅ SetOkHandler: Wrap data to {code:0, data}           │
│  ✅ SetErrorHandler: Map errors to {code, message}      │
│  ✅ Format responses consistently                       │
│  ✅ Log errors (optional)                               │
└─────────────────────────────────────────────────────────┘
```

### Migration Guide

#### From errorx.Wrapf to BizError

**Before:**
```go
// Logic layer
return nil, errorx.Wrapf(nil, "用户不存在")  // Always returns 400

// Response
HTTP/1.1 400 Bad Request
用户不存在
```

**After:**
```go
// Define error
var ErrUserNotFound = errors.NewBizError(404, 10001, "用户不存在")

// Logic layer
return nil, errors.ErrUserNotFound  // Returns 404

// Global handler
httpx.SetErrorHandler(func(err error) (int, interface{}) {
    if bizErr, ok := err.(*errors.BizError); ok {
        return bizErr.HTTPCode, &types.Response{
            Code:    bizErr.BizCode,
            Message: bizErr.Message,
        }
    }
    // ...
})

// Response
HTTP/1.1 404 Not Found
{"code":10001,"message":"用户不存在","data":null}
```

### Testing

#### Unit Test for Logic Layer

```go
func TestCreateOrder_InsufficientStock(t *testing.T) {
    // Setup mock
    mockProductModel := &MockProductModel{
        FindOneFunc: func(ctx context.Context, id int64) (*model.Product, error) {
            return &model.Product{
                Id:    1,
                Stock: 5, // Only 5 items in stock
            }, nil
        },
    }
    
    svcCtx := &svc.ServiceContext{ProductModel: mockProductModel}
    logic := NewCreateOrderLogic(context.Background(), svcCtx)
    
    // Execute with quantity = 10 (exceeds stock)
    _, err := logic.CreateOrder(&types.CreateOrderRequest{
        ProductId: 1,
        Quantity:  10,
    })
    
    // Assert
    assert.Error(t, err)
    bizErr, ok := err.(*errors.BizError)
    assert.True(t, ok)
    assert.Equal(t, errors.CodeStockNotEnough, bizErr.BizCode)
    assert.Equal(t, http.StatusBadRequest, bizErr.HTTPCode)
}
```

#### Integration Test for Error Handler

```go
func TestErrorHandler_BizError(t *testing.T) {
    handler := func(err error) (int, interface{}) {
        if bizErr, ok := err.(*errors.BizError); ok {
            return bizErr.HTTPCode, &types.Response{
                Code:    bizErr.BizCode,
                Message: bizErr.Message,
            }
        }
        return http.StatusInternalServerError, nil
    }
    
    // Test user not found error
    code, body := handler(errors.ErrUserNotFound)
    assert.Equal(t, http.StatusNotFound, code)
    
    resp := body.(*types.Response)
    assert.Equal(t, 10001, resp.Code)
    assert.Equal(t, "User not found", resp.Message)
}
```

---

## References

- [Custom HTTP Status Codes in go-zero](./custom-http-status-codes.md)
- [REST API Patterns](./rest-api-patterns.md)
- [go-zero Official Documentation](https://go-zero.dev)
- [HTTP Status Codes (RFC 7231)](https://tools.ietf.org/html/rfc7231#section-6)
- [Problem Details for HTTP APIs (RFC 7807)](https://tools.ietf.org/html/rfc7807)

---

## Changelog

- **2024-01-01**: Initial version
- Comprehensive guide for e-commerce response handling
- Business error code design patterns
- Real-world examples with order creation flow
