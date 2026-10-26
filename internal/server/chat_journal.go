package server

import (
	"errors"
	"strings"
	"time"

	"github.com/wnddd839/codebuddy-proxy/internal/gateway"
	"github.com/wnddd839/codebuddy-proxy/internal/openai"
	"github.com/wnddd839/codebuddy-proxy/internal/provider"
	"github.com/wnddd839/codebuddy-proxy/internal/strutil"
)

func (s *Server) newProxyRequestID() string {
	return strutil.RandomHex(16)
}

func (s *Server) recordChatJournal(
	started time.Time,
	proxyID string,
	opts gateway.CompleteOptions,
	publicModel string,
	stream bool,
	ok bool,
	errMsg string,
	usage provider.Usage,
	result *gateway.CompleteResult,
) {
	if s == nil || s.Svc == nil {
		return
	}
	input := gateway.ChatRecordInput{
		Started:        started,
		ProxyRequestID: proxyID,
		SessionKey:     opts.SessionKey,
		SessionLabel:   opts.SessionLabel,
		Model:          strutil.First(publicModel, opts.Model, "auto"),
		Stream:         stream,
		Ok:             ok,
		ErrMsg:         strings.TrimSpace(errMsg),
		Usage:          usage,
	}
	if result != nil {
		input.Trace = result.Trace
		input.Account = result.Account
		input.AccountID = result.AccountID
	}
	s.Svc.RecordChat(input)
}

// logChatFailure 把聊天失败写进活动日志（报错日志）。用量明细已单独落盘，
// 这里补上错误现场：阶段、类型、上游原因与错误码，方便直接定位是哪一跳挂了。
// 请求体过大、JSON 解析失败等本地校验错误由调用方自行忽略，避免噪声。
func (s *Server) logChatFailure(model, stream, stage, errMsg string) {
	if s == nil || strings.TrimSpace(errMsg) == "" {
		return
	}
	typ, code := openai.ClassifyUpstream(errors.New(errMsg))
	s.LogActivity("chat-error", map[string]any{
		"model":       strutil.First(strings.TrimSpace(model), "auto"),
		"stream":      stream == "stream",
		"stage":       stage,
		"type":        typ,
		"cause":       openai.ClassifyCause(errors.New(errMsg)),
		"error":       strutil.Truncate(errMsg, 600),
		"retry_after": openai.RetryAfter(errors.New(errMsg)),
		"code":        code,
	})
}
