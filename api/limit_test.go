package api_test

import (
	"auditlimit/api"
	"errors"
	"testing"
	"time"

	"github.com/gogf/gf/v2/os/gctx"
)

// 合法的限流值应被原样采用。
func TestGetVisitorWithModel(t *testing.T) {
	t.Setenv("GPT-9_9-RATE-UNIT-TEST", "7/24h")

	limit, per, limiter, err := api.GetVisitorWithModel(gctx.New(), "token-rate", "gpt-9.9-rate-unit-test")
	if err != nil {
		t.Fatalf("err = %v, 期望 nil", err)
	}
	if limit != 7 || per != 24*time.Hour {
		t.Fatalf("limit/per = %d/%s, 期望 7/24h0m0s", limit, per)
	}
	if limiter == nil {
		t.Fatal("limiter 为 nil")
	}
}

// 非法的限流值不能 panic(历史上 "0/1h" 会让 rate.NewLimiter 内部 per/limit 整数除零),
// 应统一回退到兜底的 40/3h。
func TestGetVisitorWithModelInvalidRateFallback(t *testing.T) {
	cases := map[string]string{
		"次数为零":  "0/1h",
		"次数为负":  "-1/1h",
		"次数非数字": "abc/3h",
		"缺少斜杠":  "40",
		"斜杠过多":  "40/3h/x",
		"时间缺单位": "40/3",
		"时长为零":  "40/0s",
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GPT-9_9-BAD-RATE-UNIT-TEST", value)

			limit, per, limiter, err := api.GetVisitorWithModel(gctx.New(), "token-bad-rate", "gpt-9.9-bad-rate-unit-test")
			if err != nil {
				t.Fatalf("err = %v, 期望 nil", err)
			}
			if limit != 40 || per != 3*time.Hour {
				t.Fatalf("limit/per = %d/%s, 期望兜底值 40/3h0m0s", limit, per)
			}
			if limiter == nil {
				t.Fatal("limiter 为 nil")
			}
		})
	}
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
