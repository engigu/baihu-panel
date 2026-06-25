package tunnel

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"github.com/hashicorp/yamux"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	EnableCompression: true, // 启用协议级别的 Deflate 数据压缩
}

func HandleTunnel(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(401, gin.H{"error": "missing authorization"})
		return
	}

	tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if tokenStr == "" {
		c.JSON(401, gin.H{"error": "invalid token format"})
		return
	}

	settingsSvc := services.NewSettingsService()
	siteConfig := settingsSvc.GetSection(constant.SectionInterconnect)
	role := siteConfig[constant.KeyInterconnectRole]

	if role != constant.InterconnectRoleMaster {
		c.JSON(403, gin.H{"error": "interconnect master role not enabled on this server"})
		return
	}

	var node models.InterconnectNode
	if err := database.DB.Where("token = ?", tokenStr).First(&node).Error; err != nil {
		c.JSON(401, gin.H{"error": "invalid interconnect token"})
		return
	}

	nodeID := node.ID

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("[Tunnel] 升级隧道连接失败 (NodeID: %s): %v", nodeID, err)
		return
	}

	// 将 WebSocket 包装为标准的 net.Conn
	netConn := NetConn(conn)

	// 获取客户端真实 IP 和端口以更新连接地址，方便面板显示
	clientIP := c.ClientIP()
	port := "unknown"
	if parts := strings.Split(netConn.RemoteAddr().String(), ":"); len(parts) > 1 {
		port = parts[len(parts)-1]
	}
	realURL := "tunnel://" + clientIP + ":" + port

	if node.URL != realURL {
		database.DB.Model(&node).Update("url", realURL)
	}

	// 我们是主节点（父节点），主动发起对子节点的请求。
	// 因此在 Yamux 协议中我们扮演 Client，而子节点扮演 Server。
	conf := yamux.DefaultConfig()
	conf.EnableKeepAlive = true

	session, err := yamux.Client(netConn, conf)
	if err != nil {
		logger.Errorf("[Tunnel] 启动 Yamux Client 失败 (NodeID: %s): %v", nodeID, err)
		netConn.Close()
		return
	}

	AddSession(nodeID, tokenStr, session)
	logger.Infof("[Tunnel] 已建立来自子节点的隧道连接: %s (%s)", nodeID, node.Name)

	go func(s *yamux.Session) {
		<-s.CloseChan()
		logger.Infof("[Tunnel] 子节点隧道连接已断开 (NodeID: %s, IsClosed: %v)", nodeID, s.IsClosed())
		RemoveSession(nodeID, s)
	}(session)
}
