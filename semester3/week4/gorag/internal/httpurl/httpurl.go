// Package httpurl builds child endpoints from configured service base URLs
// without discarding a path prefix or query parameters.
package httpurl

import (
	"net/url"
	"path"
	"strings"
)

// JoinPath returns a copy of base with suffix joined onto base's path. Base
// query parameters and userinfo are preserved; fragments are dropped.
func JoinPath(base *url.URL, suffix string) *url.URL {
	clone := *base
	basePath := strings.TrimRight(clone.Path, "/")
	joined := path.Join(basePath, suffix)
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	clone.Path = joined
	clone.Fragment = ""
	return &clone
}

// OpenAICompatible joins a provider path while honoring the common convention
// that a base path already ending in "/v1" is the API root.
func OpenAICompatible(base *url.URL, suffix string) *url.URL {
	basePath := strings.TrimRight(base.Path, "/")
	if strings.HasSuffix(basePath, "/v1") {
		return JoinPath(base, suffix)
	}
	return JoinPath(base, "v1/"+suffix)
}
