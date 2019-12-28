package middleware

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Tracing adds opentracing support for outgoing calls
func Tracing(uri string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*http.Request)
			if !ok {
				logger.Warnf("[Tracing] request not http request: %v", req)
			} else if req == nil {
				logger.Warnf("[Tracing] nil request: %v", req)
			} else {
				var span opentracing.Span
				span, ctx = opentracing.StartSpanFromContext(ctx, uri, ext.SpanKindRPCClient)
				defer span.Finish()
				span.SetTag("http.target", req.Host)
				ext.HTTPMethod.Set(span, req.Method)

				span.Tracer().Inject(
					span.Context(),
					opentracing.HTTPHeaders,
					opentracing.HTTPHeadersCarrier(req.Header),
				)
			}

			return next(ctx, request)
		}
	}

}
