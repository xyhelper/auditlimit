package config

import (
	"context"
	"os"

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

// GetStringWithEnv 读取配置项,优先级为: 环境变量 > 配置文件 > 默认值。
//
// 为什么不直接使用 g.Cfg().MustGetWithEnv:
// gogf/gf v2.10 起 GetWithEnv 会先用 utils.FormatCmdKey 把键名转成小写再去配置文件里查找,
// 而本项目的 config.yaml 使用与 README 环境变量同名的大写键(如 GPT-4O),小写化后无法命中,
// 会导致 config.yaml 中的配置被静默忽略。这里按原始键名显式查找,保证两种来源都能生效。
//
// 另外这里使用 Get 而不是 MustGet,因为在没有配置文件时(例如 Docker 镜像内)MustGet 会 panic。
func GetStringWithEnv(ctx context.Context, key string, def ...string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if v, err := g.Cfg().Get(ctx, key); err == nil && v != nil && !v.IsNil() {
		if s := v.String(); s != "" {
			return s
		}
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
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
