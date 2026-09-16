package api

import (
	"auditlimit/config"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"golang.org/x/time/rate"
)

// ErrModelDisabled 表示该模型被配置为禁用(值为 config.DisabledValue)。
// 调用方据此返回 403, 而不是当作内部错误返回 500。
var ErrModelDisabled = errors.New("model disabled")

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	Per      time.Duration
}

// 限流配置非法时使用的兜底限流, 与历史上的硬编码兜底值保持一致。
const (
	defaultLimit = 40
	defaultPer   = 3 * time.Hour
)

// parseModelRate 解析 "次数/时间" 形式的限流值, 例如 "20/3h"、"7/24h"。
// 次数必须是正整数, 时长必须是可解析的正 duration, 否则返回 ok=false 交由调用方兜底。
//
// 为什么必须校验次数 > 0: rate.NewLimiter 内部有 per/limit 的整数除法,
// limit 为 0 时(显式写 "0/1h", 或次数写成非数字而被 gconv.Int 转成 0)会直接 panic。
func parseModelRate(value string) (limit int, per time.Duration, ok bool) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0, false
	}
	limit = gconv.Int(parts[0])
	if limit <= 0 {
		return 0, 0, false
	}
	per, err := time.ParseDuration(parts[1])
	if err != nil || per <= 0 {
		return 0, 0, false
	}
	return limit, per, true
}

var visitors = make(map[string]*visitor)
var mtx sync.Mutex

func GetVisitor(key string, limit int, per time.Duration) *rate.Limiter {
	mtx.Lock()
	defer mtx.Unlock()

	v, exists := visitors[key]
	if !exists {
		limiter := rate.NewLimiter(rate.Every(per/time.Duration(limit)), limit)
		visitors[key] = &visitor{limiter, time.Now(), per}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func GetVisitorWithModel(ctx g.Ctx, token, model string) (limit int, per time.Duration, limiter *rate.Limiter, err error) {
	// 模型被配置为禁用时直接拒绝, 不进入限流逻辑, 也不会消耗额度。
	// 这里必须前置判断: "DISABLED" 不含 "/", 若继续往下走会被当成格式错误而回退到兜底限流, 反而放行了该模型。
	if config.IsModelDisabled(ctx, model) {
		return 0, 0, nil, ErrModelDisabled
	}
	// 模型名统一按 NormalizeKey 的约定映射为配置键(转大写, "." 换成 "_"),
	// 否则形如 gpt-5.6-sol-wm 的模型名会因 "." 被当作配置路径分隔符而取不到配置。
	modelKey := config.NormalizeKey(model)
	modelrate := config.GetModelRate(ctx, model)
	if modelrate == "" {
		modelrate = config.GetStringWithEnv(ctx, "DEFAULT")
	}
	limit, per, ok := parseModelRate(modelrate)
	if !ok {
		// 值为空是"未配置"的常态(例如 Docker 部署未挂载 config.yaml 且未设环境变量),
		// 静默回退即可, 避免每个请求都刷一条告警; 只有写了值却写错才需要提醒。
		if modelrate != "" {
			// 配置值非法不该把该模型打成 500(会连累主业务), 统一回退到兜底限流并告警。
			g.Log().Warningf(ctx, "限流值 %q 非法(应形如 \"次数/时间\", 次数为正整数且时长为合法正 duration), 模型 %q 回退到 %d/%s", modelrate, model, defaultLimit, defaultPer)
		}
		limit, per = defaultLimit, defaultPer
	}
	return limit, per, GetVisitor(token+"|"+modelKey, limit, per), nil
}

func CleanupVisitors() {
	mtx.Lock()
	defer mtx.Unlock()

	for token, v := range visitors {
		if time.Since(v.lastSeen) > v.Per {
			delete(visitors, token)
		}
	}
}

func init() {
	// 每星期清理一次
	go func() {
		for {
			time.Sleep(time.Hour * 24 * 7)
			CleanupVisitors()
		}
	}()
}
