package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/why444216978/go-util/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/why444216978/gin-api/library/jaeger"
)

type MDReaderWriter struct {
	metadata.MD
}

// ForeachKey implements ForeachKey of opentracing.TextMapReader
func (c MDReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vs := range c.MD {
		for _, v := range vs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set implements Set() of opentracing.TextMapWriter
func (c MDReaderWriter) Set(key, val string) {
	key = strings.ToLower(key)
	c.MD[key] = append(c.MD[key], val)
}

// ClientInterceptor grpc client
func ClientInterceptor(spanContext opentracing.SpanContext) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if assert.IsNil(jaeger.Tracer) {
			return errors.New("jaeger client is nil")
		}

		span := opentracing.StartSpan(
			"call gRPC",
			opentracing.ChildOf(spanContext),
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC"},
			ext.SpanKindRPCClient,
		)

		defer span.Finish()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		err := jaeger.Tracer.Inject(span.Context(), opentracing.TextMap, MDReaderWriter{md})
		if err != nil {
			span.LogFields(log.String("inject-error", err.Error()))
		}

		newCtx := metadata.NewOutgoingContext(ctx, md)
		err = invoker(newCtx, method, req, reply, cc, opts...)
		if err != nil {
			span.LogFields(log.String("call-error", err.Error()))
		}
		return err
	}
}
