package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
)

// Cleanup is a middleware guarding the call and doing some cleanup
func Cleanup() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (resp interface{}, err error) {
			req, _ := request.(*http.Request)
			uri := req.RequestURI
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("[Cleanup] panic when handing %s:%v", uri, r)
					err = fmt.Errorf("panic error: %v", r)
				}
			}()
			resp, err = next(ctx, request)
			return
		}
	}
}
