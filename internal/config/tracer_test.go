package config

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
)

func TestTracer(t *testing.T) {
	cfg := New()
	cfg.Tracing.On = true
	cfg.Tracing.Addr = "127.0.0.1:6831"
	tracer, err := NewTracer(cfg)
	assert.NoError(t, err)

	span := tracer.StartSpan("wukongchat_root")
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	r1 := foo3(ctx, "Hello 测试中文")
	r2 := foo4(ctx, "Hello foo4")
	fmt.Println(r1, r2)
	span.Finish()

	time.Sleep(1 * time.Second)

}
func foo3(ctx context.Context, req string) (reply string) {
	//1.创建子span
	span, _ := opentracing.StartSpanFromContext(ctx, "span_foo3")
	defer func() {
		//4.接口调用完，在tag中设置request和reply
		span.SetTag("request", req)
		span.SetTag("reply", reply)
		span.Finish()
	}()

	println(req)
	//2.模拟处理耗时
	time.Sleep(time.Second / 2)
	//3.返回reply
	reply = "foo3Reply"
	return
}

// 跟foo3一样逻辑
func foo4(ctx context.Context, req string) (reply string) {
	span, _ := opentracing.StartSpanFromContext(ctx, "span_foo4")
	defer func() {
		span.SetTag("request", req)
		span.SetTag("reply", reply)
		span.Finish()
	}()

	foo5("foo5", span.Context())

	println(req)
	time.Sleep(time.Second / 2)
	reply = "foo4Reply"
	return
}

// 跟foo3一样逻辑
func foo5(req string, parentCtx opentracing.SpanContext) (reply string) {
	span := opentracing.StartSpan("span_foo5", opentracing.ChildOf(parentCtx))
	defer func() {
		span.SetTag("request", req)
		span.SetTag("reply", reply)
		span.Finish()
	}()

	println(req)
	time.Sleep(time.Second / 2)
	reply = "foo5Reply"
	return
}
