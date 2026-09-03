package voiceecho

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startEchoServer 起一个进程内回声服务器，返回其 URL。
func startEchoServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(NewWSSHandler())
	t.Cleanup(server.Close)
	return server
}

// dial 建立 WebSocket 连接并注册关闭清理。
func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// readMessage 带超时读一条消息。
func readMessage(t *testing.T, conn *websocket.Conn) (int, []byte) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	msgType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return msgType, payload
}

func TestEchoBinaryFrame(t *testing.T) {
	server := startEchoServer(t)
	conn := dial(t, "ws"+server.URL[len("http"):])

	// 模拟 12B 帧头 + 负载的音频包。
	packet := append([]byte{
		0x01, 0x00, 0x00, 0x00, // seq = 1
		0xE8, 0x03, 0x00, 0x00, // timestamp_ms = 1000
		0x04, 0x00, // size = 4
		0x01, 0x00, // flags = gapBefore
	}, 0xAA, 0xBB, 0xCC, 0xDD...)

	if err := conn.WriteMessage(websocket.BinaryMessage, packet); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	msgType, payload := readMessage(t, conn)
	if msgType != websocket.BinaryMessage {
		t.Fatalf("msgType = %d, want binary", msgType)
	}
	if !bytes.Equal(payload, packet) {
		t.Fatalf("echo mismatch: got % x, want % x", payload, packet)
	}
}

func TestEchoTextControl(t *testing.T) {
	server := startEchoServer(t)
	conn := dial(t, "ws"+server.URL[len("http"):])

	control := `{"type":"finish","idempotencyKey":"test-key-0001"}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(control)); err != nil {
		t.Fatalf("write text: %v", err)
	}
	msgType, payload := readMessage(t, conn)
	if msgType != websocket.TextMessage {
		t.Fatalf("msgType = %d, want text", msgType)
	}
	if string(payload) != control {
		t.Fatalf("echo mismatch: got %s, want %s", payload, control)
	}
}

func TestEchoManyFramesInOrder(t *testing.T) {
	server := startEchoServer(t)
	conn := dial(t, "ws"+server.URL[len("http"):])

	const total = 16
	for seq := 0; seq < total; seq++ {
		if err := conn.WriteMessage(websocket.BinaryMessage, []byte{byte(seq)}); err != nil {
			t.Fatalf("write %d: %v", seq, err)
		}
	}
	for seq := 0; seq < total; seq++ {
		_, payload := readMessage(t, conn)
		if len(payload) != 1 || payload[0] != byte(seq) {
			t.Fatalf("echo order broken at %d: got % x", seq, payload)
		}
	}
}
