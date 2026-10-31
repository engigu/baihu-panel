package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDemoGuard_Disabled(t *testing.T) {
	constant.DemoMode = false

	r := gin.New()
	r.POST("/test", DemoGuard("演示模式禁止"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("预期 200 OK，实际状态码: %d", w.Code)
	}
}

func TestDemoGuard_Enabled_HTTP(t *testing.T) {
	constant.DemoMode = true
	defer func() { constant.DemoMode = false }()

	r := gin.New()
	r.POST("/test", DemoGuard("演示模式下禁止修改配置"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("预期 400 BadRequest，实际状态码: %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "演示模式下禁止修改配置") {
		t.Fatalf("响应体未包含预期的错误提示: %s", body)
	}
}

func TestDemoGuard_Enabled_SSE(t *testing.T) {
	constant.DemoMode = true
	defer func() { constant.DemoMode = false }()

	r := gin.New()
	r.POST("/test", DemoGuard("演示模式禁止部署应用"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/test?stream=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("预期 Content-Type 为 text/event-stream，实际: %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: error") || !strings.Contains(body, "演示模式禁止部署应用") {
		t.Fatalf("SSE 事件未包含预期的 error 结构: %s", body)
	}
}

func TestDemoGuard_Enabled_WebSocket(t *testing.T) {
	constant.DemoMode = true
	defer func() { constant.DemoMode = false }()

	r := gin.New()
	r.GET("/ws", DemoGuard("终端功能已禁用"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket 拨号失败: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("预期状态码 101 Switching Protocols，实际: %d", resp.StatusCode)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("读取拦截提示消息失败: %v", err)
	}

	if !strings.Contains(string(msg), "终端功能已禁用") {
		t.Fatalf("WebSocket 消息中未包含拦截提示: %s", string(msg))
	}
}
