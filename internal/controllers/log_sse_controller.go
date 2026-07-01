package controllers

import (
	"fmt"
	"io"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type LogSSEController struct{}

func NewLogSSEController() *LogSSEController {
	return &LogSSEController{}
}

func (lc *LogSSEController) StreamLog(c *gin.Context) {
	logIDStr := c.Query("log_id")
	if logIDStr == "" {
		c.JSON(400, gin.H{"error": "log_id is required"})
		return
	}

	logID := logIDStr

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	// c.Header("Access-Control-Allow-Origin", "*")

	// 1. 检查数据库中是否已结束
	var taskLog models.TaskLog
	res := database.DB.Where("id = ?", logID).Limit(1).Find(&taskLog)
	if res.Error == nil && res.RowsAffected > 0 {
		if taskLog.Status != "running" {
			// 已结束，读取库内日志
			content, err := utils.DecompressFromBase64(string(taskLog.Output))
			if err != nil {
				c.SSEvent("message", gin.H{"text": "解压日志失败: " + err.Error()})
				c.Writer.Flush()
				return
			}
			c.SSEvent("message", gin.H{"text": content})
			c.Writer.Flush()
			return
		}
	}

	// 2. 未结束或未找到记录，尝试从 TinyLogManager 获取
	tl := tasks.GetActiveLog(logID)
	if tl == nil {
		c.SSEvent("message", gin.H{"text": "未找到正在运行的任务日志"})
		c.Writer.Flush()
		return
	}

	// 发送系统提示
	c.SSEvent("message", gin.H{"text": fmt.Sprintf("[System] 连接成功，正在监听日志... (LogID: %s)\n", logID)})
	c.Writer.Flush()

	// 发送最后 100 行
	lastLines, err := tl.ReadLastLines(100)
	if err == nil && len(lastLines) > 0 {
		c.SSEvent("message", gin.H{"text": string(lastLines)})
		c.Writer.Flush()
	}

	// 订阅实时更新
	sub := tl.Subscribe()
	defer tl.Unsubscribe(sub)

	// 推送更新
	c.Stream(func(w io.Writer) bool {
		select {
		case data, ok := <-sub:
			if !ok {
				// 任务结束，尝试刷新最后一次库内完整内容
				var finalLog models.TaskLog
				res := database.DB.Where("id = ?", logID).Limit(1).Find(&finalLog)
				if res.Error == nil && res.RowsAffected > 0 {
					content, _ := utils.DecompressFromBase64(string(finalLog.Output))
					if content != "" {
						c.SSEvent("message", gin.H{"text": "\n--- 任务已结束 ---\n"})
					}
				}
				return false
			}
			c.SSEvent("message", gin.H{"text": string(data)})
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
