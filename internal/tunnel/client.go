package tunnel

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

var (
	tunnelActive bool
	clientConn   *yamux.Session
	clientMu     sync.Mutex
	stopCh       chan struct{}

	// LocalEngine 保存全局的 Gin 引擎或 HTTP 处理器，用于接收主节点的隧道请求并进行纯内存函数路由
	LocalEngine http.Handler
)

// SetLocalEngine 由 bootstrap 层注入
func SetLocalEngine(engine http.Handler) {
	LocalEngine = engine
}

// StartClient 尝试连接到父节点面板（如果已配置）
func StartClient() {
	clientMu.Lock()
	if tunnelActive {
		clientMu.Unlock()
		return
	}
	tunnelActive = true
	stopCh = make(chan struct{})
	clientMu.Unlock()

	go runClientLoop()
}

// StopClient 停止后台的隧道客户端
func StopClient() {
	clientMu.Lock()
	if !tunnelActive {
		clientMu.Unlock()
		return
	}
	tunnelActive = false
	if stopCh != nil {
		close(stopCh)
	}
	if clientConn != nil {
		clientConn.Close()
	}
	clientMu.Unlock()
}

// Init 从数据库加载配置并初始化对应的后台隧道服务
func Init() {
	siteConfig := services.NewSettingsService().GetSection(constant.SectionInterconnect)
	role := siteConfig[constant.KeyInterconnectRole]
	ApplyRole(role)
}

// ApplyRole 根据互联角色动态启停背景协程服务
func ApplyRole(role string) {
	switch role {
	case constant.InterconnectRoleMaster:
		// 主控角色：无需后台轮询，等待子节点上报即可
		StopClient()
	case constant.InterconnectRoleChild:
		// 子节点：关闭可能存在的主控会话，启动连接守护
		CloseAllSessions()
		StartClient()
	default:
		// 未开启或离线
		StopClient()
		CloseAllSessions()
	}
}

// IsTunnelConnected 返回当前子节点隧道是否已连接到主节点
func IsTunnelConnected() bool {
	clientMu.Lock()
	defer clientMu.Unlock()
	if !tunnelActive || clientConn == nil {
		return false
	}
	return !clientConn.IsClosed()
}

func runClientLoop() {
	settingsSvc := services.NewSettingsService()
	
	const (
		initialBackoff = 5 * time.Second
		maxBackoff     = 300 * time.Second
		fatalBackoff   = 60 * time.Second
	)
	
	backoff := initialBackoff
	retryCount := 0
	var lastErrorMsg string

	for {
		clientMu.Lock()
		if !tunnelActive {
			clientMu.Unlock()
			return
		}
		clientMu.Unlock()

		siteConfig := settingsSvc.GetSection(constant.SectionInterconnect)

		role := siteConfig[constant.KeyInterconnectRole]
		if role != constant.InterconnectRoleChild {
			time.Sleep(10 * time.Second)
			continue
		}

		parentURL := siteConfig[constant.KeyInterconnectParentURL]
		parentToken := siteConfig[constant.KeyInterconnectParentToken]

		if parentURL == "" || parentToken == "" {
			time.Sleep(10 * time.Second)
			continue
		}

		u, err := url.Parse(parentURL)
		if err != nil {
			logger.Errorf("[Tunnel] 无效的主节点 URL: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		scheme := "ws"
		if u.Scheme == "https" {
			scheme = "wss"
		}

		wsURL := fmt.Sprintf("%s://%s/api/v1/interconnect/tunnel", scheme, u.Host)

		header := http.Header{}
		header.Set("Authorization", "Bearer "+parentToken)

		dialer := &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
		}
		dialer.EnableCompression = true
		if u.Scheme == "https" {
			dialer.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}

		conn, resp, err := dialer.Dial(wsURL, header)
		if err != nil {
			retryCount++
			errMsg := err.Error()

			// 特殊处理致命的认证或配置错误 (仅401作为致命错误，403使用指数退避)
			isFatal := resp != nil && resp.StatusCode == 401
			if isFatal {
				backoff = fatalBackoff
			}

			// 先计算退避时间 (加上 0-8 秒随机抖动防止风暴)
			jitter := time.Duration(rand.Intn(8000)) * time.Millisecond
			sleepDuration := backoff + jitter

			// 日志记录
			if retryCount == 1 || errMsg != lastErrorMsg {
				logger.Errorf("[Tunnel] 连接主节点隧道失败: %v, 下次重试将在 %v 后", err, sleepDuration.Round(time.Second))
			} else {
				logger.Warnf("[Tunnel] 仍无法连接主节点，已尝试 %d 次, 下次重试将在 %v 后", retryCount, sleepDuration.Round(time.Second))
			}
			
			lastErrorMsg = errMsg

			time.Sleep(sleepDuration)
			
			// 指数级增加退避时间 (最大不超过 maxBackoff)
			if !isFatal && backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}

		// 连接成功，重置退避状态
		retryCount = 0
		lastErrorMsg = ""
		backoff = initialBackoff

		logger.Infof("[Tunnel] 已成功连接到主节点隧道: %s", wsURL)

		netConn := NetConn(conn)
		trackedConn := &trafficCounterConn{Conn: netConn}
		conf := yamux.DefaultConfig()
		conf.EnableKeepAlive = true

		// 我们是子节点，接受来自主节点的请求。
		// 因此在这里，我们是 Yamux Server，负责监听传入的流。
		session, err := yamux.Server(trackedConn, conf)
		if err != nil {
			logger.Errorf("[Tunnel] 启动 Yamux Server 失败: %v", err)
			trackedConn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// 如果已经存在上一个活跃的隧道连接，在重建前必须将其显式 Close()，以释放该会话在后台运行的 HTTP 监听与监听协程
		clientMu.Lock()
		if clientConn != nil {
			clientConn.Close()
		}
		clientConn = session
		clientMu.Unlock()

		services.GetSystemWSManager().Broadcast(constant.EventInterconnectChildStatus, map[string]interface{}{
			"connected": true,
		})

		// 启动子节点主动上报服务
		StartReporter(parentURL, parentToken)

		// 启动本地 HTTP 代理服务 (阻塞直到会话关闭)
		serveLocalProxy(session)
		
		// 断开时停止上报
		StopReporter()

		session.Close()
		services.GetSystemWSManager().Broadcast(constant.EventInterconnectChildStatus, map[string]interface{}{
			"connected": false,
		})
		logger.Infof("[Tunnel] 已从主节点隧道断开连接")
		
		// 断开后稍微缓冲一下再去重试
		time.Sleep(2 * time.Second)
	}
}


