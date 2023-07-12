package config

import (
	"context"
	"io"
	"net/http"

	llog "github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

// Tracer唐僧叨叨Tracer
type Tracer struct {
	jaegerconfig jaegercfg.Configuration
	cfg          *Config
	tracer       opentracing.Tracer
	closer       io.Closer
	llog.Log
}

// NewTracer NewTracer
func NewTracer(cfg *Config) (*Tracer, error) {
	if !cfg.Tracing.On {
		return &Tracer{
			cfg: cfg,
		}, nil
	}
	t := &Tracer{
		cfg: cfg,
		Log: llog.NewTLog("Tracer"),
	}
	t.jaegerconfig = jaegercfg.Configuration{
		ServiceName: cfg.AppID,
		Headers: &jaeger.HeadersConfig{
			TraceContextHeaderName: "wukongchat-trace-id",
		},
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: cfg.Tracing.Addr,
		},
	}
	var err error
	t.tracer, t.closer, err = t.jaegerconfig.New(cfg.AppID, jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(t.tracer)
	return t, nil
}

// Close 关闭
func (t *Tracer) Close() error {
	if t.closer != nil {
		return t.closer.Close()
	}
	return nil
}

// StartSpanFromContext StartSpanFromContext
func (t *Tracer) StartSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	if !t.cfg.Tracing.On {
		return &EmptySpan{}, ctx
	}
	return opentracing.StartSpanFromContext(ctx, operationName, opts...)
}

// StartSpan StartSpan
func (t *Tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	if !t.cfg.Tracing.On {
		return &EmptySpan{}
	}
	return t.tracer.StartSpan(operationName, opts...)
}

// ContextWithSpan ContextWithSpan
func (t *Tracer) ContextWithSpan(ctx context.Context, span opentracing.Span) context.Context {
	return opentracing.ContextWithSpan(ctx, span)
}

// Inject Inject
func (t *Tracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	if !t.cfg.Tracing.On {
		return nil
	}
	return t.tracer.Inject(sm, format, carrier)
}

// InjectHTTPHeader InjectHTTPHeader
func (t *Tracer) InjectHTTPHeader(sm opentracing.SpanContext, header http.Header) error {
	if !t.cfg.Tracing.On {
		return nil
	}
	return t.tracer.Inject(sm, opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(header))
}

// Extract Extract
func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	if !t.cfg.Tracing.On {
		return &EmptySpanContext{}, nil
	}
	return t.tracer.Extract(format, carrier)
}

// ExtractHTTPHeader ExtractHTTPHeader
func (t *Tracer) ExtractHTTPHeader(header http.Header) (opentracing.SpanContext, error) {
	return t.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(header))
}

// GinMiddle gin的中间件
func (t *Tracer) GinMiddle() gin.HandlerFunc {
	return func(c *gin.Context) {
		var spanContext opentracing.SpanContext
		if t.cfg.Tracing.On {
			var parentSpan opentracing.Span
			spCtx, err := t.ExtractHTTPHeader(c.Request.Header)
			if err != nil {
				//t.Warn("从http header里获取span失败！", zap.Error(err))
				parentSpan, _ = t.StartSpanFromContext(context.Background(), c.FullPath())
			} else {
				parentSpan = t.StartSpan(
					c.FullPath(),
					opentracing.ChildOf(spCtx),
					opentracing.Tag{Key: string(ext.Component), Value: "HTTP"},
				)
			}
			defer parentSpan.Finish()

			spanContext = parentSpan.Context()
		} else {
			spanContext = &EmptySpanContext{}
		}
		c.Set("spanContext", spanContext)

		c.Next()
		if t.cfg.Tracing.On {
			err := t.InjectHTTPHeader(spanContext, c.Request.Header)
			if err != nil {
				t.Warn("注入tracing的http header失败！", zap.Error(err))
			}
		}
	}
}

// EmptySpan 空的span
type EmptySpan struct {
}

// Finish Finish
func (e *EmptySpan) Finish() {

}

// FinishWithOptions FinishWithOptions
func (e *EmptySpan) FinishWithOptions(opts opentracing.FinishOptions) {

}

// Context Context
func (e *EmptySpan) Context() opentracing.SpanContext {
	return nil
}

// SetOperationName SetOperationName
func (e *EmptySpan) SetOperationName(operationName string) opentracing.Span {
	return e
}

// SetTag SetTag
func (e *EmptySpan) SetTag(key string, value interface{}) opentracing.Span {
	return e
}

// LogFields LogFields
func (e *EmptySpan) LogFields(fields ...log.Field) {

}

// LogKV LogKV
func (e *EmptySpan) LogKV(alternatingKeyValues ...interface{}) {

}

// SetBaggageItem SetBaggageItem
func (e *EmptySpan) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	return e
}

// BaggageItem BaggageItem
func (e *EmptySpan) BaggageItem(restrictedKey string) string {
	return ""
}

// Tracer Tracer
func (e *EmptySpan) Tracer() opentracing.Tracer {
	return nil
}

// LogEvent LogEvent
func (e *EmptySpan) LogEvent(event string) {

}

// LogEventWithPayload LogEventWithPayload
func (e *EmptySpan) LogEventWithPayload(event string, payload interface{}) {

}

// Log Log
func (e *EmptySpan) Log(data opentracing.LogData) {

}

// EmptySpanContext EmptySpanContext
type EmptySpanContext struct {
}

// ForeachBaggageItem ForeachBaggageItem
func (e *EmptySpanContext) ForeachBaggageItem(handler func(k, v string) bool) {

}
