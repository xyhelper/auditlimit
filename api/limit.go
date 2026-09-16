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
	modelratearr := strings.Split(modelrate, "/")
	// g.Dump(modelratearr)
	if len(modelratearr) != 2 {
		modelratearr = []string{"40", "3h"}
	}
	limit = gconv.Int(modelratearr[0])
	// per = gconv.Duration(modelratearr[1])
	per, err = time.ParseDuration(modelratearr[1])
	if err != nil {
		return 0, 0, nil, err
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
