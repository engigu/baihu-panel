package controllers

import (
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type SystemWSController struct {
	manager *services.SystemWSManager
}

func NewSystemWSController() *SystemWSController {
	return &SystemWSController{
		manager: services.GetSystemWSManager(),
	}
}

func (sc *SystemWSController) HandleEvents(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("[SystemWS] 升级 WebSocket 失败: %v", err)
		return
	}

	client := sc.manager.Register(conn)
	defer sc.manager.Unregister(client)

	// 启动写循环
	go sc.writeLoop(client)

	// 启动读循环 (主要用于检测连接断开和维持心跳)
	sc.readLoop(client)
}

func (sc *SystemWSController) readLoop(client *services.ClientConnection) {
	defer client.Close()

	client.Conn.SetReadLimit(constant.MaxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(constant.PongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(constant.PongWait))
		return nil
	})

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warnf("[SystemWS] 客户端异常断开: %v", err)
			}
			break
		}
		// 暂时不处理来自前端的消息，前端仅作为接收方
	}
}

func (sc *SystemWSController) writeLoop(client *services.ClientConnection) {
	ticker := time.NewTicker(constant.PingPeriod)
	defer func() {
		ticker.Stop()
		client.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道关闭
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
