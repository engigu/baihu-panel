package tunnel

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/logger"
	
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/yamux"
)

// ProxyHTTP 以完全透明的流式转发将 HTTP 请求代理到远程节点 (主节点调用)
func ProxyHTTP(nodeID string, c *gin.Context, targetPath string) error {
	sess := GetSession(nodeID)
	if sess == nil {
		return errors.New("node is offline or tunnel not established")
	}

	if sess.Token == "" {
		return errors.New("node token not found in session cache")
	}

	// 获取真实客户端 IP，传递给子节点用于审计日志等
	clientIP := c.ClientIP()

	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = "tunnel.local"
		req.URL.Path = targetPath
		// gin 和 httputil 默认会保留并透传 Query 参数

		req.Header.Set("Authorization", "Bearer "+sess.Token)
		// 删除父节点的 Cookie 避免干扰子节点的会话验证
		req.Header.Del("Cookie")

		// 强制加上该 Header，防止出现节点相互代理的死循环
		req.Header.Set("X-Tunnel-Proxy", "true")
		
		// 传递真实的客户端 IP 给子节点
		if clientIP != "" {
			req.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: sess.Transport, // 直接复用 AddSession 时创建的全局连接池，由原生库处理 Context 取消
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Errorf("[Tunnel] 节点 %s 代理隧道请求发生错误: %v", nodeID, err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("Tunnel connection error: " + err.Error()))
		},
	}

	proxy.ServeHTTP(c.Writer, c.Request)
	return nil
}

// serveLocalProxy 本地服务代理逻辑 (子节点调用)
func serveLocalProxy(session *yamux.Session) {
	if LocalEngine == nil {
		logger.Errorf("[Tunnel] 无法启动本地代理服务，因为 LocalEngine 尚未注入。")
		return
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 安全规则 1：防止互相调用的死循环
		if strings.HasPrefix(r.URL.Path, "/api/v1/interconnect/proxy/") {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Recursive proxy loops are not allowed via tunnel"))
			return
		}

		// 安全规则 2：严格限制只能访问本机的 /api/v1/ 接口
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Tunnel access is strictly restricted to /api/v1/ endpoints"))
			return
		}

		// 确保保留 X-Tunnel-Proxy 请求头，以防后续逻辑需要判定
		r.Header.Set("X-Tunnel-Proxy", "true")

		// 纯内存函数调用，直接扔给 Gin 引擎，零网络开销！
		LocalEngine.ServeHTTP(w, r)
	})

	server := &http.Server{
		Handler:     handler,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 15 * time.Second,
	}

	// 监听会话关闭信号，主动关闭 http.Server 以彻底释放协程
	go func() {
		<-session.CloseChan()
		server.Close()
	}()

	// yamux.Session 实现了标准的 net.Listener 接口！
	err := server.Serve(session)
	if err != nil && err != http.ErrServerClosed && err != yamux.ErrSessionShutdown {
		logger.Errorf("[Tunnel] Yamux 代理服务意外停止: %v", err)
	}
}
