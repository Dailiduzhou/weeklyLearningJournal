package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
)

func NewTestContext(method, path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c
}

func NewTestContextWithParams(method, path string, params map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	if params != nil {
		for k, v := range params {
			c.Params = append(c.Params, gin.Param{Key: k, Value: v})
		}
	}
	return c
}

func ExecuteMiddleware(c *gin.Context, handler gin.HandlerFunc) {
	handler(c)
}

type TestResponseRecorder struct {
	*httptest.ResponseRecorder
}

func NewTestResponseRecorder() *TestResponseRecorder {
	return &TestResponseRecorder{
		ResponseRecorder: httptest.NewRecorder(),
	}
}
