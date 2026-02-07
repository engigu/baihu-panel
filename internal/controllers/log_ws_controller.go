package controllers

import (
	"fmt"
	"strconv"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type LogWSController struct{}

func NewLogWSController() *LogWSController {
	return &LogWSController{}
}

func (lc *LogWSController) StreamLog(c *gin.Context) {
	logIDStr := c.Query("log_id")
	if logIDStr == "" {
		return
	}

	logID, err := strconv.ParseUint(logIDStr, 10, 32)
	if err != nil {
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 1. 检查数据库中是否已结束
	var taskLog models.TaskLog
	if err := database.DB.First(&taskLog, uint(logID)).Error; err == nil {
		if taskLog.Status != "running" {
			// 已结束，读取库内日志
			content, err := utils.DecompressFromBase64(taskLog.Output)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte("解压日志失败: "+err.Error()))
				return
			}
			conn.WriteMessage(websocket.TextMessage, []byte(content))
			return
		}
	}

	// 2. 未结束或未找到记录，尝试从 TinyLogManager 获取
	tl := tasks.GetActiveLog(uint(logID))
	if tl == nil {
		conn.WriteMessage(websocket.TextMessage, []byte("未找到正在运行的任务日志"))
		return
	}

	// 发送系统提示
	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("[System] 连接成功，正在监听日志... (LogID: %d)\n", logID)))

	// 发送最后 100 行
	lastLines, err := tl.ReadLastLines(100)
	if err == nil && len(lastLines) > 0 {
		conn.WriteMessage(websocket.TextMessage, lastLines)
	}

	// 订阅实时更新
	sub := tl.Subscribe()
	defer tl.Unsubscribe(sub)

	// 推送更新
	for {
		select {
		case data, ok := <-sub:
			if !ok {
				// 任务结束，尝试刷新最后一次库内完整内容
				var finalLog models.TaskLog
				if err := database.DB.First(&finalLog, uint(logID)).Error; err == nil {
					content, _ := utils.DecompressFromBase64(finalLog.Output)
					if content != "" {
						conn.WriteMessage(websocket.TextMessage, []byte("\n--- 任务已结束 ---\n"))
						// 这里可以选择性再推一次完整版，或直接退出
					}
				}
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}
