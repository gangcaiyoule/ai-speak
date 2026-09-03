// Package voiceecho 提供 R6 回包链路的 WSS 音频回声端点。
//
// 回声协议 v0：对收到的每条 WebSocket 消息原样回传——
//
//   - 二进制消息 = 12 字节帧头 + PCM 负载（见 mobile 侧 FrameHeaderCodec），
//     客户端据此验证帧头编解码与弱网字段（seq/timestamp_ms/size/flags）；
//   - 文本消息 = JSON 控制帧（start/finish/cancel），客户端会话层据此
//     驱动生命周期终态回执。
//
// 该端点只用于协议打通与联调，不承载识别/评测；接真实云服务时由
// 对应协议实现替换，回声语义不在生产路径上。
package voiceecho

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader 把 HTTP 连接升级为 WebSocket。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// 回声端点是联调用途，无浏览器跨域语义；部署到公网前由网关收紧 Origin。
	CheckOrigin: func(*http.Request) bool { return true },
}

// NewWSSHandler 返回逐条回显消息的 WebSocket 处理器。
//
// 客户端连接后：每收到一条消息（任意类型）立即原样回传；读侧出错或
// 连接关闭则结束。单连接单 goroutine，无共享状态。
//
// Returns the http.HandlerFunc for mounting on the router.
func NewWSSHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			msgType, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(msgType, payload); err != nil {
				return
			}
		}
	}
}
