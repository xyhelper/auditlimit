package api

import (
	"net/http"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
)

// 响应体中 detail.code 的取值, 客户端据此区分响应类型。
// 取值与 README 的「超速返回格式」「禁用返回格式」保持一致。
const (
	// CodeModelCapExceeded 429, 该模型的使用额度已耗尽。
	CodeModelCapExceeded = "model_cap_exceeded"
	// CodeModelDisabled 403, 该模型被配置为 DISABLED。
	CodeModelDisabled = "model_disabled"
	// CodeFlaggedByModeration 400, 内容审核判定为违规。
	CodeFlaggedByModeration = "flagged_by_moderation"
)

// writeModelDisabled 返回模型被禁用的响应。
// 用 403 加独立的 code, 便于客户端与本项目日志区分"被禁用"与"触发限流"两种情况。
func writeModelDisabled(r *ghttp.Request, model string) {
	r.Response.Status = http.StatusForbidden
	r.Response.WriteJson(g.Map{
		"detail": g.Map{
			"code":    CodeModelDisabled,
			"message": "The model " + model + " is disabled.\n" + "模型 " + model + " 已被禁用,当前不可使用,请更换其他模型后重试.",
		},
	})
}

// writeModerationRejected 返回内容审核未通过的响应(400)。
func writeModerationRejected(r *ghttp.Request) {
	r.Response.Status = http.StatusBadRequest
	r.Response.WriteJson(g.Map{
		"detail": g.Map{
			"code":    CodeFlaggedByModeration,
			"message": "This content may violate [OpenAI Usage Policies](https://openai.com/policies/usage-policies).",
		},
	})
}

// writeTooManyRequests 返回触发限流的响应(429), 结构见 README「超速返回格式」:
//
//	{"detail":{"clears_in":252,"code":"model_cap_exceeded","message":"..."}}
//
// clearsIn 为预计需要等待的秒数; 预留失败(已超出突发额度)而无法给出具体时长时传 0。
func writeTooManyRequests(r *ghttp.Request, model string, limit int, per time.Duration, clearsIn int) {
	limitStr := gconv.String(limit)
	perStr := gconv.String(per)
	var en, zh string
	if clearsIn > 0 {
		wait := gconv.String(clearsIn)
		en = "You have triggered the usage frequency limit of " + model + ", the current limit is " + limitStr + " times/" + perStr + ", please wait " + wait + " seconds before trying again."
		zh = "您已经触发 " + model + " 使用频率限制,当前限制为 " + limitStr + " 次/" + perStr + ",请等待 " + wait + " 秒后再试."
	} else {
		en = "You have triggered the usage frequency limit of " + model + ", the current limit is " + limitStr + " times/" + perStr + ", please wait a moment before trying again."
		zh = "您已经触发 " + model + " 使用频率限制,当前限制为 " + limitStr + " 次/" + perStr + ",请稍后再试."
	}
	r.Response.Status = http.StatusTooManyRequests
	r.Response.WriteJson(g.Map{
		"detail": g.Map{
			"clears_in": clearsIn,
			"code":      CodeModelCapExceeded,
			"message":   en + "\n" + zh,
		},
	})
}
