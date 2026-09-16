package config_test

import (
	"testing"

	"auditlimit/config"
	"github.com/gogf/gf/v2/os/gctx"
)

func TestNormalizeKey(t *testing.T) {
	cases := map[string]string{
		"gpt-4o":                      "GPT-4O",
		"GPT-4O":                      "GPT-4O",
		"gpt-4":                       "GPT-4",
		"text-davinci-002-render-sha": "TEXT-DAVINCI-002-RENDER-SHA",
		// 含 "." 的模型名, "." 需替换为 "_"
		"gpt-5.6-sol-wm": "GPT-5_6-SOL-WM",
		"gpt-5.6.1-x":    "GPT-5_6_1-X",
		"o4-mini-high":   "O4-MINI-HIGH",
	}
	for in, want := range cases {
		if got := config.NormalizeKey(in); got != want {
			t.Errorf("NormalizeKey(%q) = %q, 期望 %q", in, got, want)
		}
	}
}

// 含 "." 的模型名应能通过 NormalizeKey 后的键名读到配置。
func TestGetModelRateWithDottedModelName(t *testing.T) {
	const key = "GPT-9_9-DOTTED-UNIT-TEST"
	t.Setenv(key, "7/1h")

	ctx := gctx.New()
	if got := config.GetModelRate(ctx, "gpt-9.9-dotted-unit-test"); got != "7/1h" {
		t.Fatalf("GetModelRate = %q, 期望 %q", got, "7/1h")
	}
}

// 兼容直接使用含 "." 原始模型名的环境变量。
func TestGetModelRateWithRawDottedKey(t *testing.T) {
	const key = "GPT-8.8-RAW-DOTTED-UNIT-TEST"
	t.Setenv(key, "9/2h")

	ctx := gctx.New()
	if got := config.GetModelRate(ctx, "gpt-8.8-raw-dotted-unit-test"); got != "9/2h" {
		t.Fatalf("GetModelRate = %q, 期望 %q", got, "9/2h")
	}
}

// 未配置的模型返回空字符串,交给调用方回退到 DEFAULT。
func TestGetModelRateNotConfigured(t *testing.T) {
	ctx := gctx.New()
	if got := config.GetModelRate(ctx, "gpt-0.0-not-configured-unit-test"); got != "" {
		t.Fatalf("GetModelRate = %q, 期望空字符串", got)
	}
}

// 禁用标记忽略大小写与首尾空白,只有写成 DISABLED 才算禁用。
func TestIsDisabled(t *testing.T) {
	for _, v := range []string{"DISABLED", "disabled", "Disabled", "  DISABLED  "} {
		if !config.IsDisabled(v) {
			t.Errorf("IsDisabled(%q) = false, 期望 true", v)
		}
	}
	for _, v := range []string{"", "0/1h", "60/3h", "DISABLE", "DISABLED1"} {
		if config.IsDisabled(v) {
			t.Errorf("IsDisabled(%q) = true, 期望 false", v)
		}
	}
}

// 模型自身的值写成 DISABLED 时判定为禁用; 未配置的模型回退到 DEFAULT, 不应被判定为禁用。
func TestIsModelDisabled(t *testing.T) {
	t.Setenv("GPT-9_9-DISABLED-UNIT-TEST", "DISABLED")

	ctx := gctx.New()
	if !config.IsModelDisabled(ctx, "gpt-9.9-disabled-unit-test") {
		t.Fatal("已禁用的模型 IsModelDisabled = false, 期望 true")
	}
	if config.IsModelDisabled(ctx, "gpt-0.0-not-configured-unit-test") {
		t.Fatal("未配置的模型 IsModelDisabled = true, 期望 false")
	}
}
