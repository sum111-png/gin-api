package jaeger

import (
	"fmt"
	"gin-api/libraries/logging"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
)

func OpenTracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := opentracing.GlobalTracer()

		_, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		if err == nil {
			return
		}

		fmt.Println("opentracing:start")
		sp := tracer.StartSpan(c.Request.RequestURI)
		span := spanContextToJaegerContext(sp.Context())
		sp.SetTag(FIELD_TRACE_ID, span.TraceID().String())
		sp.SetTag(FIELD_SPAN_ID, span.SpanID().String())
		sp.SetTag(FIELD_LOG_ID, logging.ValueLogID(c))

		c.Set(FIELD_TRACER, tracer)
		c.Set(FIELD_SPAN, sp)
		defer sp.Finish()

		c.Next()
	}
}
