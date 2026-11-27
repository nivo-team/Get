package ui

import (
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"sheep-get/internal/clipboard"
)

// clipboardReadTimeout 是一次剪贴板读取愿意等多久。监视器按固定间隔轮询，读不到就当
// 「这一轮没有新链接」，不让一次卡住的读取把后面的轮次全部堵死。
const clipboardReadTimeout = 2 * time.Second

// scopeClipboard 是剪贴板监视器看到的系统剪贴板：NSGo 的读取是异步回调，且必须在应用
// 自己的反应式图里发起，这里把它接成监视器需要的同步读。
//
// 读取失败（作用域已关闭、平台拒绝、超时）一律报「没有文本」：剪贴板监听是后台行为，
// 读不到不该影响任何下载。
type scopeClipboard struct {
	scope *core.Scope
}

func newScopeClipboard(scope *core.Scope) *scopeClipboard {
	return &scopeClipboard{scope: scope}
}

// Text 读一次剪贴板文本。
func (c *scopeClipboard) Text() (string, bool) {
	if c == nil || c.scope == nil {
		return "", false
	}
	result := make(chan string, 1)
	c.scope.Post(func() {
		c.scope.ReadClipboard(func(text string) {
			select {
			case result <- text:
			default:
			}
		})
	})
	select {
	case text := <-result:
		return text, text != ""
	case <-time.After(clipboardReadTimeout):
		return "", false
	}
}

var _ clipboard.Reader = (*scopeClipboard)(nil)
