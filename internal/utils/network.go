package utils

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CheckWSOrigin 校验 WebSocket 的 Origin 来源是否安全。
// 默认仅允许同源请求，可通过环境变量 BH_ALLOWED_ORIGINS 配置额外的允许列表（逗号分隔）。
func CheckWSOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// 非浏览器发起的请求（如直接用脚本连接）通常不带 Origin，默认放行。
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// 0. 开发环境校验：如果是非 Release 模式，默认放行
	if gin.Mode() != gin.ReleaseMode {
		return true
	}

	// 1. 同源校验：Origin 的 Host 与请求头中的 Host 一致
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}

	// 容错处理：如果因为 Nginx 配置了 $host 而丢掉了端口，导致一方带端口一方不带端口时，尝试忽略端口进行域名比对
	uHostOnly := u.Hostname()
	rHostOnly := r.Host
	if h, _, err := net.SplitHostPort(r.Host); err == nil {
		rHostOnly = h
	}
	if strings.EqualFold(uHostOnly, rHostOnly) {
		return true
	}

	// 2. 环境变量配置的允许列表校验
	allowedOrigins := os.Getenv("BH_ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		origins := strings.Split(allowedOrigins, ",")
		for _, o := range origins {
			o = strings.TrimSpace(o)
			if o == "*" {
				return true
			}
			// 匹配完整 Origin (如 http://localhost:5173) 或仅 Host 部分
			if strings.EqualFold(o, origin) || strings.EqualFold(o, u.Host) {
				return true
			}
		}
	}

	// 3. 允许来自 localhost、127.0.0.1 以及局域网内网 IP 的请求 (方便本地开发、虚拟机调试和局域网部署)
	hostname := u.Hostname()
	if strings.HasPrefix(hostname, "localhost") || strings.HasPrefix(hostname, "127.0.0.1") ||
		strings.HasPrefix(hostname, "192.168.") || strings.HasPrefix(hostname, "10.") ||
		strings.HasPrefix(hostname, "172.") {
		return true
	}

	return false
}
