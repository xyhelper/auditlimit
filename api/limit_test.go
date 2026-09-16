package api_test

import (
	"auditlimit/api"
	"errors"
	"testing"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

func TestGetVisitorWithModel(t *testing.T) {
	ctx := gctx.New()
	limit, per, limiter, err := api.GetVisitorWithModel(ctx, "token", "text-davinci-002-render-sha")
	if err != nil {
		g.Log().Error(ctx, "GetVisitorWithModel", err)
		return
	}
	g.Dump(limiter)
	g.Log().Info(ctx, "limit:", limit, "per:", per, "limiter:", limiter)

}

// 模型的值被配置为 DISABLED 时, 应返回 ErrModelDisabled, 而不是当成限流格式错误。
func TestGetVisitorWithModelDisabled(t *testing.T) {
	t.Setenv("GPT-9_9-DISABLED-UNIT-TEST", "DISABLED")

	ctx := gctx.New()
	_, _, _, err := api.GetVisitorWithModel(ctx, "token", "gpt-9.9-disabled-unit-test")
	if !errors.Is(err, api.ErrModelDisabled) {
		t.Fatalf("err = %v, 期望 %v", err, api.ErrModelDisabled)
	}
}
