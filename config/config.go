package config

import (
	"context"
	"os"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/util/gconv"
)

var (
	PORT = 8080
	// PlusModels     = garray.NewStrArrayFrom([]string{"gpt-4", "gpt-4o", "gpt-4-browsing", "gpt-4-plugins", "gpt-4-mobile", "gpt-4-code-interpreter", "gpt-4-dalle", "gpt-4-gizmo", "gpt-4-magic-create", "gpt-4o-canmore"})
	// O1Models       = garray.NewStrArrayFrom([]string{"o1-preview", "o1-mini"})
	ForbiddenWords = []string{} // 禁止词
	// LIMIT          = 40                 // 限制次数
	// PER            = time.Hour * 3      // 限制时间
	// O1LIMIT        = 5                  // 限制次数
	// O1PER          = time.Hour * 24 * 7 // 限制时间
	OAIKEY    = "" // OAIKEY
	OAIKEYLOG = "" // OAIKEYLOG 隐藏
	// MODERATION     = "https://api.openai.com/v1/moderations" // OPENAI Moderation 检测
	MODERATION = "https://gateway.ai.cloudflare.com/v1/040ac2002b4dd67637e97c628feb3484/xyhelper/openai/moderations"
)

// NormalizeKey 将模型名转换为配置键,约定为: 转大写,并把 "." 替换为 "_"。
//
// 为什么必须把 "." 换成 "_":
//  1. gf 的配置查找把 "." 当作路径分隔符,键 GPT-5.6 会被拆成 GPT-5 下的 6,永远取不到值;
//  2. "." 不是合法的环境变量名字符,shell 里无法直接 export,只能靠 env 命令变通。
//
// 因此模型 gpt-5.6-sol-wm 对应的配置键是 GPT-5_6-SOL-WM,
// 在 config.yaml 与环境变量中都按这个名字书写。
func NormalizeKey(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, ".", "_"))
}

// configFileValue 从配置文件中按原始键名精确读取,
// 第二个返回值表示该键在配置文件中是否存在(存在但值为空字符串时也返回 true)。
func configFileValue(ctx context.Context, key string) (string, bool) {
	if v, err := g.Cfg().Get(ctx, key); err == nil && v != nil && !v.IsNil() {
		return v.String(), true
	}
	// 键中含 "." 时会被 g.Cfg().Get 当成路径解析而查不到,
	// 此时退回平铺数据做精确匹配,以便兼容在配置里直接书写原始模型名(如 GPT-5.6-SOL-WM)的写法。
	if !strings.Contains(key, ".") {
		return "", false
	}
	if data, err := g.Cfg().Data(ctx); err == nil {
		if raw, ok := data[key]; ok {
			return gconv.String(raw), true
		}
	}
	return "", false
}

// GetStringWithEnv 读取字符串配置项,优先级为: 配置文件 > 环境变量 > 默认值。
//
// 为什么不直接使用 g.Cfg().MustGetWithEnv:
// gogf/gf v2.10 起 GetWithEnv 会先用 utils.FormatCmdKey 把键名转成小写再去配置文件里查找,
// 而本项目的 config.yaml 使用与 README 环境变量同名的大写键(如 GPT-4O),小写化后无法命中,
// 会导致 config.yaml 中的配置被静默忽略。这里按原始键名显式查找,保证两种来源都能生效。
//
// 另外这里使用 Get 而不是 MustGet,因为在没有配置文件时(例如 Docker 镜像内)MustGet 会 panic。
func GetStringWithEnv(ctx context.Context, key string, def ...string) string {
	// 配置文件优先: 只要配置文件中存在该键就直接采用,即使其值为空字符串。
	// 这与旧版 gf 的 GetWithEnv 行为一致,即配置文件里显式留空的项不会被同名环境变量覆盖。
	if s, ok := configFileValue(ctx, key); ok {
		return s
	}
	// 仅在配置文件中不存在该键时,才回退到环境变量。
	if v := os.Getenv(key); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

// getStringWithEnvAny 依次用多个候选键查找,优先级同样为 配置文件 > 环境变量 > 默认值。
// 先把所有候选键在配置文件中过一遍,再统一看环境变量,避免环境变量比配置文件里的次选键更优先。
func getStringWithEnvAny(ctx context.Context, keys []string, def ...string) string {
	for _, key := range keys {
		if s, ok := configFileValue(ctx, key); ok {
			return s
		}
	}
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

// GetModelRate 读取某个模型的限流配置(形如 "40/3h")。
//
// 配置键名约定见 NormalizeKey,例如 gpt-5.6-sol-wm 对应 GPT-5_6-SOL-WM。
// 为了兼容历史配置,同时接受直接书写含 "." 的原始模型名(如 GPT-5.6-SOL-WM)。
// 返回空字符串表示该模型未配置。
func GetModelRate(ctx context.Context, model string) string {
	keys := []string{NormalizeKey(model)}
	if upper := strings.ToUpper(model); upper != keys[0] {
		keys = append(keys, upper)
	}
	return getStringWithEnvAny(ctx, keys)
}

// DisabledValue 是限流配置里的特殊值, 表示禁止用户使用该模型。
// 例如在 config.yaml 里写 `GPT-5_5-WM: "DISABLED"`,
// 或在环境变量里写 `GPT-5_5-WM=DISABLED`, 都会禁止用户使用模型 gpt-5.5-wm。
const DisabledValue = "DISABLED"

// IsDisabled 判断限流配置项的值是否为禁用标记, 忽略大小写与首尾空白。
func IsDisabled(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), DisabledValue)
}

// IsModelDisabled 判断某个模型是否被配置为禁用。
//
// 模型自身没有配置限流时会看 DEFAULT, 因此把 DEFAULT 写成 DISABLED
// 可以一次性禁用所有未单独配置的模型。
func IsModelDisabled(ctx context.Context, model string) bool {
	rate := GetModelRate(ctx, model)
	if rate == "" {
		rate = GetStringWithEnv(ctx, "DEFAULT")
	}
	return IsDisabled(rate)
}

func init() {
	ctx := gctx.GetInitCtx()
	port := gconv.Int(GetStringWithEnv(ctx, "PORT"))
	if port > 0 {
		PORT = port
	}
	g.Log().Info(ctx, "PORT:", PORT)
	// limit := g.Cfg().MustGetWithEnv(ctx, "LIMIT").Int()
	// if limit > 0 {
	// 	LIMIT = limit
	// }
	// g.Log().Info(ctx, "LIMIT:", LIMIT)
	// per := g.Cfg().MustGetWithEnv(ctx, "PER").Duration()
	// if per > 0 {
	// 	PER = per
	// }
	// g.Log().Info(ctx, "PER:", PER)
	// o1limit := g.Cfg().MustGetWithEnv(ctx, "O1LIMIT").Int()
	// if o1limit > 0 {
	// 	O1LIMIT = o1limit
	// }
	// g.Log().Info(ctx, "O1LIMIT:", O1LIMIT)
	// o1per := g.Cfg().MustGetWithEnv(ctx, "O1PER").Duration()
	// if o1per > 0 {
	// 	O1PER = o1per
	// }
	oaikey := GetStringWithEnv(ctx, "OAIKEY")
	// oaikey 不为空
	if oaikey != "" {
		OAIKEY = oaikey
		// 日志隐藏 oaikey，有 * 代表有值
		OAIKEYLOG = "******"
	}
	g.Log().Info(ctx, "OAIKEY:", OAIKEYLOG)
	moderation := GetStringWithEnv(ctx, "MODERATION")
	if moderation != "" {
		MODERATION = moderation
	}
	g.Log().Info(ctx, "MODERATION:", MODERATION)
}
