package api

import (
	"auditlimit/config"
	"errors"
	"strings"
	"time"

	"github.com/gogf/gf/v2/container/garray"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func AuditLimit(r *ghttp.Request) {
	ctx := r.Context()
	// 获取Bearer Token 用来判断用户身份
	token := r.Header.Get("Authorization")
	// 移除 "Bearer " 前缀。
	// 不能像旧版那样无条件截掉前 7 个字符: 头部长度不足 7 时会 panic 成 500,
	// 不带前缀的 token 也会被截错, 使不同用户碰撞到同一份额度。
	if len(token) > 7 && strings.EqualFold(token[:7], "bearer ") {
		token = token[7:]
	}
	g.Log().Debug(ctx, "token", token)
	// 获取gfsessionid 可以用来分析用户是否多设备登录
	gfsessionid := r.Cookie.Get("gfsessionid").String()
	g.Log().Debug(ctx, "gfsessionid", gfsessionid)
	// 获取referer 可以用来判断用户请求来源
	referer := r.Header.Get("referer")
	g.Log().Debug(ctx, "referer", referer)
	// 获取请求内容
	reqJson, err := r.GetJson()
	if err != nil {
		// 必须 return: 否则会继续按空请求体往下走, 最后用 200 覆盖掉刚写好的 400。
		g.Log().Error(ctx, "GetJson", err)
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"detail": err.Error(),
		})
		return
	}
	action := reqJson.Get("action").String() // action为 next时才是真正的请求，否则可能是继续上次请求 action 为 variant 时为重新生成
	g.Log().Debug(ctx, "action", action)

	model := reqJson.Get("model").String() // 模型名称
	g.Log().Debug(ctx, "model", model)
	// system_hint := reqJson.Get("system_hints.0").String() // 系统提示
	system_hints := reqJson.Get("system_hints").Strings() // 系统提示
	systemHints := garray.NewStrArrayFrom(system_hints)

	g.Log().Debug(ctx, "systemHints", systemHints)
	prompt := reqJson.Get("messages.0.content.parts.0").String() // 输入内容
	g.Log().Debug(ctx, "prompt", prompt)

	// 研究 / 代理模式会用 system_hints 覆盖模型名,
	// 提前覆盖, 使后面的禁用判断与限流都以最终生效的模型名为准。
	if systemHints.Contains("research") {
		model = "research"
	}
	if systemHints.Contains("agent") {
		model = "agent"
	}

	// 模型被配置为禁用时直接拒绝, 不消耗额度, 也不做内容审核
	if config.IsModelDisabled(ctx, model) {
		g.Log().Info(ctx, "model disabled", model)
		writeModelDisabled(r, model)
		return
	}

	// 判断提问内容是否包含禁止词
	if containsAny(ctx, prompt, config.ForbiddenWords) {
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"detail": "请珍惜账号,不要提问违禁内容.",
		})
		return
	}

	// OPENAI Moderation 检测
	if config.OAIKEY != "" && prompt != "" {
		// 检测是否包含违规内容
		respVar := g.Client().SetHeaderMap(g.MapStrStr{
			"Authorization": "Bearer " + config.OAIKEY,
			"Content-Type":  "application/json",
		}).PostVar(ctx, config.MODERATION, g.Map{
			"input": prompt,
			"model": "omni-moderation-latest",
		})

		// 返回的 json 中 results.flagged 为 true 时为违规内容
		// respBody := resp.ReadAllString()
		//g.Log().Debug(ctx, "resp:", respBody)
		g.Dump(respVar)
		respJson := gjson.New(respVar)
		// 这里是**有意为之**的 fail-open 设计: 审核接口不可达、返回非 JSON 或缺少 results 字段时,
		// 下面的取值会得到 false, 即按"通过"放行。目的是不让审核服务的异常连带影响主业务;
		// 改成 fail-closed 会导致上游抖动时整个服务都不可用。排障请看上面的 g.Dump 输出。
		isFlagged := respJson.Get("results.0.flagged").Bool()
		g.Log().Debug(ctx, "flagged", isFlagged)
		if isFlagged {
			writeModerationRejected(r)
			return
		}
	}
	limit, per, limiter, err := GetVisitorWithModel(ctx, token, model)
	if err != nil {
		// 正常情况下禁用的模型已在前面拦下, 这里兜底, 避免漏网时被当成 500 内部错误。
		if errors.Is(err, ErrModelDisabled) {
			g.Log().Info(ctx, "model disabled", model)
			writeModelDisabled(r, model)
			return
		}
		g.Log().Error(ctx, "GetVisitorWithModel", err)
		r.Response.Status = 500
		r.Response.WriteJson(g.Map{
			"detail": err.Error(),
		})
		return
	}
	// 获取剩余次数
	remain := limiter.TokensAt(time.Now())
	g.Log().Debug(ctx, token, model, "remain", remain, "limit", limit, "per", per)
	if remain < 1 {
		reservation := limiter.ReserveN(time.Now(), 1)
		if !reservation.OK() {
			// 处理预留失败的情况(已超出突发额度), 此时无法给出具体等待时间
			writeTooManyRequests(r, model, limit, per, 0)
			return
		}
		delayFrom := reservation.Delay()
		reservation.Cancel() // 取消预留，不消耗令牌

		g.Log().Debug(ctx, "delayFrom", delayFrom)
		writeTooManyRequests(r, model, limit, per, int(delayFrom.Seconds()))
		return
	}
	// 消耗一个令牌
	limiter.Allow()

	r.Response.Status = 200

}

// 判断字符串是否包含数组中的任意一个元素
func containsAny(ctx g.Ctx, text string, array []string) bool {
	for _, item := range array {
		if strings.Contains(text, item) {
			g.Log().Debug(ctx, "containsAny", text, item)
			return true
		}
	}
	return false
}
