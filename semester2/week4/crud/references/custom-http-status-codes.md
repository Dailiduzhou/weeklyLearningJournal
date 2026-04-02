# Custom HTTP Status Codes in go-zero

## Overview

This guide covers best practices for implementing custom HTTP status codes in go-zero REST APIs, including error handling patterns and practical implementation examples.

## go-zero Error Handling Foundation

### errorx Package Capabilities

The `github.com/zeromicro/go-zero/core/errorx` package provides:

```go
// Available functions
func Wrap(err error, message string) error              // Wrap error with message
func Wrapf(err error, format string, args ...any) error // Wrap with formatted message
func Chain(fns ...func() error) error                   // Chain multiple error checks
func In(err error, errs ...error) bool                  // Check if error is in list

// Available types
type AtomicError struct{ ... }  // Thread-safe atomic error
type BatchError struct{ ... }   // Batch error collection
```

**Important:** `errorx` does **NOT** provide `NewCodeError` or any built-in HTTP status code mapping.

### Default Error Behavior

Without custom configuration, go-zero's default error handler in `httpx`:

```go
// From github.com/zeromicro/go-zero/rest/httpx/responses.go
func doHandleError(...) {
    if handler == nil {
        if errcode.IsGrpcError(err) {
            // gRPC errors → gRPC status codes
            http.Error(w, err.Error(), errcode.CodeFromGrpcError(err))
        } else {
            // All other errors → 400 Bad Request
            http.Error(w, err.Error(), http.StatusBadRequest)
        }
        return
    }
    
    // With custom handler → custom status codes
    code, body := handler(err)
    writeJson(w, code, body)
}
```

**Default behavior:**
- ✅ gRPC errors → Mapped to appropriate HTTP codes
- ❌ All other errors → `400 Bad Request` (incorrect for 404, 401, 403, 500, etc.)

---

## Solution Comparison

| Approach | go-zero Support | Implementation Effort | Maintainability | Recommendation |
|----------|----------------|----------------------|-----------------|----------------|
| **Global Error Handler** | ✅ `httpx.SetErrorHandler` | Low (define error constants) | High (centralized) | ⭐⭐⭐⭐⭐ Best for most projects |
| **CodeError Type** | ✅ `httpx.SetErrorHandler` | Medium (define custom type) | High (type-safe) | ⭐⭐⭐⭐ Best for rich error details |
| **Handler Control** | ✅ `httpx.WriteJsonCtx` | Low (no setup needed) | Low (scattered logic) | ⭐⭐ Only for special cases |

---

## Solution 1: Global Error Handler ⭐ **Recommended**

### When to Use

- Most go-zero projects
- Unified error response format
- Centralized error code management
- Multiple routes sharing the same error types

### Implementation

#### 1. Define Business Errors

```go
// internal/errors/errors.go
package errors

import "errors"

var (
    // User errors
    ErrUserNotFound     = errors.New("user not found")
    ErrUserDisabled     = errors.New("user account disabled")
    
    // Authentication errors
    ErrUnauthorized     = errors.New("unauthorized")
    ErrTokenExpired     = errors.New("token expired")
    ErrInvalidToken     = errors.New("invalid token")
    
    // Permission errors
    ErrForbidden        = errors.New("permission denied")
    ErrAdminOnly        = errors.New("admin access required")
    
    // Input validation errors
    ErrInvalidInput     = errors.New("invalid input parameters")
    ErrMissingField     = errors.New("required field missing")
    ErrInvalidFormat    = errors.New("invalid format")
    
    // Business logic errors
    ErrDuplicateEmail   = errors.New("email already exists")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrOperationFailed  = errors.New("operation failed")
)
```

#### 2. Register Global Error Handler

```go
// cmd/user.go (or main entry point)
package main

import (
    "net/http"
    
    "crud/cmd/internal/config"
    "crud/cmd/internal/errors"
    "crud/cmd/internal/handler"
    "crud/cmd/internal/svc"
    
    "github.com/zeromicro/go-zero/core/conf"
    "github.com/zeromicro/go-zero/rest"
    "github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/user.yaml", "the config file")

func main() {
    flag.Parse()
    
    var c config.Config
    conf.MustLoad(*configFile, &c)
    
    // Register custom error handler BEFORE creating server
    httpx.SetErrorHandler(func(err error) (int, any) {
        switch {
        // 404 Not Found
        case errors.Is(err, errors.ErrUserNotFound):
            return http.StatusNotFound, map[string]string{
                "code":    "USER_NOT_FOUND",
                "message": err.Error(),
            }
        
        // 400 Bad Request
        case errors.Is(err, errors.ErrInvalidInput),
             errors.Is(err, errors.ErrMissingField),
             errors.Is(err, errors.ErrInvalidFormat):
            return http.StatusBadRequest, map[string]string{
                "code":    "INVALID_INPUT",
                "message": err.Error(),
            }
        
        // 401 Unauthorized
        case errors.Is(err, errors.ErrUnauthorized),
             errors.Is(err, errors.ErrTokenExpired),
             errors.Is(err, errors.ErrInvalidToken):
            return http.StatusUnauthorized, map[string]string{
                "code":    "UNAUTHORIZED",
                "message": err.Error(),
            }
        
        // 403 Forbidden
        case errors.Is(err, errors.ErrForbidden),
             errors.Is(err, errors.ErrAdminOnly):
            return http.StatusForbidden, map[string]string{
                "code":    "FORBIDDEN",
                "message": err.Error(),
            }
        
        // 409 Conflict
        case errors.Is(err, errors.ErrDuplicateEmail):
            return http.StatusConflict, map[string]string{
                "code":    "CONFLICT",
                "message": err.Error(),
            }
        
        // 422 Unprocessable Entity
        case errors.Is(err, errors.ErrInsufficientBalance):
            return http.StatusUnprocessableEntity, map[string]string{
                "code":    "BUSINESS_ERROR",
                "message": err.Error(),
            }
        
        // 500 Internal Server Error (default)
        default:
            return http.StatusInternalServerError, map[string]string{
                "code":    "INTERNAL_ERROR",
                "message": "internal server error",
            }
        }
    })
    
    server := rest.MustNewServer(c.RestConf)
    defer server.Stop()
    
    ctx := svc.NewServiceContext(c)
    handler.RegisterHandlers(server, ctx)
    
    server.Start()
}
```

#### 3. Use in Logic Layer

```go
// internal/logic/user/detaillogic.go
func (l *DetailLogic) Detail(req *types.UserInfoRequest) (resp *types.UserInfoResp, err error) {
    user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.ID)
    if err != nil {
        if err == model.ErrNotFound {
            // Returns 404 automatically
            return nil, errors.ErrUserNotFound
        }
        // Returns 500 automatically
        return nil, err
    }
    
    return &types.UserInfoResp{
        ID:       user.Id,
        Username: user.Username,
        Role:     user.Role,
    }, nil
}

// internal/logic/user/registerlogic.go
func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResp, err error) {
    // Check if username exists
    exists, err := l.svcCtx.UserModel.ExistsByUsername(l.ctx, req.Username)
    if err != nil {
        return nil, err
    }
    if exists {
        // Returns 409 Conflict automatically
        return nil, errors.ErrDuplicateEmail
    }
    
    // Create user...
    return &types.RegisterResp{Username: req.Username}, nil
}
```

### Advantages

- ✅ **Clean three-layer architecture**: Logic returns domain errors, Handler stays thin
- ✅ **Unified error format**: All endpoints return consistent error structure
- ✅ **Easy maintenance**: Centralized error code management
- ✅ **Follows go-zero conventions**: Official recommended approach
- ✅ **Type-safe**: Using `errors.Is()` for comparison

### Disadvantages

- Global effect: All routes share the same error code mapping
- Limited flexibility: Cannot customize error format per route without additional logic

---

## Solution 2: CodeError Type Pattern

### When to Use

- Need rich error details (field names, validation errors, etc.)
- Business error codes separate from HTTP status codes
- More flexible error response structure
- Type-safe error handling

### Implementation

#### 1. Define CodeError Type

```go
// internal/errors/code_error.go
package errors

import (
    "fmt"
    "net/http"
)

// CodeError represents an error with HTTP status code and business code
type CodeError struct {
    HTTPCode int    `json:"-"`                  // HTTP status code (not exposed to client)
    Code     int    `json:"code"`               // Business error code
    Msg      string `json:"msg"`                // Error message
    Details  any    `json:"details,omitempty"` // Optional error details
}

func (e *CodeError) Error() string {
    return e.Msg
}

// NewCodeError creates a new CodeError
func NewCodeError(httpCode, code int, msg string, details ...any) *CodeError {
    ce := &CodeError{
        HTTPCode: httpCode,
        Code:     code,
        Msg:      msg,
    }
    if len(details) > 0 {
        ce.Details = details[0]
    }
    return ce
}

// Convenience constructors for common HTTP status codes
func BadRequest(code int, msg string, details ...any) *CodeError {
    return NewCodeError(http.StatusBadRequest, code, msg, details...)
}

func Unauthorized(code int, msg string) *CodeError {
    return NewCodeError(http.StatusUnauthorized, code, msg)
}

func Forbidden(code int, msg string) *CodeError {
    return NewCodeError(http.StatusForbidden, code, msg)
}

func NotFound(code int, msg string) *CodeError {
    return NewCodeError(http.StatusNotFound, code, msg)
}

func Conflict(code int, msg string, details ...any) *CodeError {
    return NewCodeError(http.StatusConflict, code, msg, details...)
}

func UnprocessableEntity(code int, msg string, details ...any) *CodeError {
    return NewCodeError(http.StatusUnprocessableEntity, code, msg, details...)
}

func InternalError(code int, msg string) *CodeError {
    return NewCodeError(http.StatusInternalServerError, code, msg)
}

// Predefined business error codes
const (
    CodeUserNotFound         = 10001
    CodeUserDisabled         = 10002
    CodeInvalidInput         = 10003
    CodeDuplicateEmail       = 10004
    CodeInsufficientBalance  = 10005
    CodeUnauthorized         = 10006
    CodeForbidden            = 10007
)
```

#### 2. Register Error Handler

```go
// cmd/user.go
func main() {
    flag.Parse()
    
    var c config.Config
    conf.MustLoad(*configFile, &c)
    
    // Register error handler for CodeError
    httpx.SetErrorHandler(func(err error) (int, any) {
        switch e := err.(type) {
        case *errors.CodeError:
            return e.HTTPCode, e
        default:
            // Fallback for non-CodeError errors
            return http.StatusInternalServerError, &errors.CodeError{
                HTTPCode: http.StatusInternalServerError,
                Code:     50000,
                Msg:      err.Error(),
            }
        }
    })
    
    server := rest.MustNewServer(c.RestConf)
    defer server.Stop()
    
    ctx := svc.NewServiceContext(c)
    handler.RegisterHandlers(server, ctx)
    
    server.Start()
}
```

#### 3. Use in Logic Layer

```go
// internal/logic/user/registerlogic.go
func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResp, err error) {
    // Validate username
    if len(req.Username) < 3 {
        return nil, errors.BadRequest(
            errors.CodeInvalidInput,
            "Username too short",
            map[string]any{
                "field":   "username",
                "min":     3,
                "current": len(req.Username),
            },
        )
    }
    
    // Check if username exists
    exists, err := l.svcCtx.UserModel.ExistsByUsername(l.ctx, req.Username)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, errors.Conflict(
            errors.CodeDuplicateEmail,
            "Username already exists",
            map[string]string{
                "field": "username",
                "value": req.Username,
            },
        )
    }
    
    // Validate password
    if len(req.Password) < 8 {
        return nil, errors.BadRequest(
            errors.CodeInvalidInput,
            "Password too short",
            map[string]any{
                "field": "password",
                "min":   8,
            },
        )
    }
    
    // Create user...
    return &types.RegisterResp{Username: req.Username}, nil
}

// internal/logic/user/detaillogic.go
func (l *DetailLogic) Detail(req *types.UserInfoRequest) (resp *types.UserInfoResp, err error) {
    user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.ID)
    if err != nil {
        if err == model.ErrNotFound {
            return nil, errors.NotFound(
                errors.CodeUserNotFound,
                fmt.Sprintf("User with ID %d not found", req.ID),
            )
        }
        return nil, err
    }
    
    return &types.UserInfoResp{
        ID:       user.Id,
        Username: user.Username,
        Role:     user.Role,
    }, nil
}
```

### Response Examples

**Bad Request with Details:**
```json
HTTP/1.1 400 Bad Request
{
    "code": 10003,
    "msg": "Username too short",
    "details": {
        "field": "username",
        "min": 3,
        "current": 2
    }
}
```

**Not Found:**
```json
HTTP/1.1 404 Not Found
{
    "code": 10001,
    "msg": "User with ID 999 not found"
}
```

**Conflict with Details:**
```json
HTTP/1.1 409 Conflict
{
    "code": 10004,
    "msg": "Username already exists",
    "details": {
        "field": "username",
        "value": "alice"
    }
}
```

### Advantages

- ✅ **Maximum flexibility**: Can carry arbitrary error details
- ✅ **Business error codes**: Separate from HTTP status codes
- ✅ **Type safety**: Compile-time error type checking
- ✅ **Semantic clarity**: Each error explicitly specifies status code
- ✅ **Rich responses**: Detailed validation errors

### Disadvantages

- Requires defining error constructor functions
- Each error needs explicit HTTP status code specification
- More boilerplate than Solution 1

---

## Solution 3: Handler-Level Control (Not Recommended)

### When to Use

- Specific route requires completely custom error response logic
- Temporary or special-case error handling
- Does NOT want to affect global error handling

### Implementation

```go
// internal/handler/user/detailhandler.go
func DetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.UserInfoRequest
        if err := httpx.Parse(r, &req); err != nil {
            httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, map[string]string{
                "code":    "PARSE_ERROR",
                "message": err.Error(),
            })
            return
        }
        
        l := user.NewDetailLogic(r.Context(), svcCtx)
        resp, err := l.Detail(&req)
        if err != nil {
            // Custom error handling per error type
            if errors.Is(err, model.ErrNotFound) {
                httpx.WriteJsonCtx(r.Context(), w, http.StatusNotFound, map[string]string{
                    "code":    "USER_NOT_FOUND",
                    "message": fmt.Sprintf("User %d not found", req.ID),
                })
            } else if strings.Contains(err.Error(), "unauthorized") {
                httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]string{
                    "code":    "UNAUTHORIZED",
                    "message": err.Error(),
                })
            } else {
                httpx.WriteJsonCtx(r.Context(), w, http.StatusInternalServerError, map[string]string{
                    "code":    "INTERNAL_ERROR",
                    "message": "internal server error",
                })
            }
            return
        }
        
        httpx.OkJsonCtx(r.Context(), w, resp)
    }
}
```

### Advantages

- ✅ **Full control**: Each route has independent error handling
- ✅ **No global impact**: Doesn't affect other routes

### Disadvantages

- ❌ **Violates three-layer architecture**: Handler contains business logic
- ❌ **Code duplication**: Every handler needs similar error handling code
- ❌ **Hard to maintain**: Error handling logic scattered across handlers
- ❌ **Not recommended** for production use

---

## Best Practices Summary

### ✅ Do

1. **Use global error handler** for most cases (Solution 1)
2. **Return domain errors** from Logic layer, not HTTP codes
3. **Use `errors.Is()`** for error comparison
4. **Define error constants** in a dedicated package
5. **Keep handlers thin** - only HTTP parsing and routing
6. **Register error handler** before creating the server
7. **Use `httpx.ErrorCtx`** instead of `httpx.Error` for context propagation

### ❌ Don't

1. **Don't put HTTP status logic** in Logic layer
2. **Don't use string matching** for error comparison (`strings.Contains`)
3. **Don't mix multiple approaches** in the same project
4. **Don't skip error handling** - always handle errors explicitly
5. **Don't return raw errors** to clients without sanitization
6. **Don't use Handler-level control** as primary approach

### Architecture Principles

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP Request                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  Handler Layer (internal/handler/)                          │
│  - Parse HTTP request                                        │
│  - Call Logic layer                                          │
│  - Return HTTP response (httpx.OkJsonCtx/ErrorCtx)          │
│  - NO business logic                                         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  Logic Layer (internal/logic/)                              │
│  - Business logic                                           │
│  - Validation                                               │
│  - Return DOMAIN errors (ErrUserNotFound, etc.)            │
│  - NO HTTP status codes                                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  Global Error Handler (httpx.SetErrorHandler)              │
│  - Map domain errors → HTTP status codes                   │
│  - Format error responses                                   │
│  - Centralized error handling                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Decision Guide

### Choose Solution 1 (Global Error Handler) when:

- ✅ Building standard REST APIs
- ✅ Need unified error responses
- ✅ Want minimal boilerplate
- ✅ Prefer centralized error management

### Choose Solution 2 (CodeError Type) when:

- ✅ Need rich error details (validation errors, field names)
- ✅ Require business error codes separate from HTTP codes
- ✅ Want maximum flexibility and type safety
- ✅ Building complex business logic APIs

### Choose Solution 3 (Handler Control) when:

- ✅ Special case for one specific route
- ✅ Temporary solution during migration
- ❌ NOT recommended as primary approach

---

## Migration Guide

### From `errorx.Wrapf` to Global Error Handler

**Before (current approach):**
```go
// Logic layer
return nil, errorx.Wrapf(nil, "用户不存在")  // Always returns 400

// Response
HTTP/1.1 400 Bad Request
用户不存在
```

**After (Solution 1):**
```go
// Define error
var ErrUserNotFound = errors.New("用户不存在")

// Logic layer
return nil, errors.ErrUserNotFound  // Returns 404

// Global handler
httpx.SetErrorHandler(func(err error) (int, any) {
    if errors.Is(err, ErrUserNotFound) {
        return http.StatusNotFound, map[string]string{"message": err.Error()}
    }
    // ...
})

// Response
HTTP/1.1 404 Not Found
{"message":"用户不存在"}
```

---

## Testing

### Test Error Responses

```go
// internal/logic/user/detaillogic_test.go
func TestDetail_UserNotFound(t *testing.T) {
    // Setup mock
    mockModel := &MockUserModel{
        FindOneFunc: func(ctx context.Context, id int64) (*model.User, error) {
            return nil, model.ErrNotFound
        },
    }
    
    svcCtx := &svc.ServiceContext{UserModel: mockModel}
    logic := NewDetailLogic(context.Background(), svcCtx)
    
    // Execute
    _, err := logic.Detail(&types.UserInfoRequest{ID: 999})
    
    // Assert
    assert.ErrorIs(t, err, errors.ErrUserNotFound)
}

// Test HTTP status code mapping
func TestErrorHandler_UserNotFound(t *testing.T) {
    handler := func(err error) (int, any) {
        if errors.Is(err, errors.ErrUserNotFound) {
            return http.StatusNotFound, map[string]string{"message": err.Error()}
        }
        return http.StatusInternalServerError, nil
    }
    
    code, body := handler(errors.ErrUserNotFound)
    assert.Equal(t, http.StatusNotFound, code)
    assert.NotNil(t, body)
}
```

---

## References

- [go-zero REST API Patterns](./rest-api-patterns.md)
- [go-zero Best Practices](../best-practices/overview.md)
- [go-zero Official Documentation](https://go-zero.dev)
- [HTTP Status Codes (RFC 7231)](https://tools.ietf.org/html/rfc7231#section-6)
