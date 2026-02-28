package tasks

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
)

type FlowNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		TaskID int `json:"taskId"` // Store the Baihu Task ID
	} `json:"data"` // VueFlow stores custom payload in data
}

type FlowEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	Label        string `json:"label"`
	SourceHandle string `json:"sourceHandle"` // Example: "success" or "failed"
	Data         struct {
		Condition string `json:"condition"`
	} `json:"data"`
}

type FlowData struct {
	Nodes []FlowNode `json:"nodes"`
	Edges []FlowEdge `json:"edges"`
}

// TriggerWorkflowNextTasks 当一个任务结束时，遍历所有开启的工作流并检查是否有满足触发条件的分支，自动触发下一级
func (es *ExecutorService) TriggerWorkflowNextTasks(taskLog *models.TaskLog) {
	// 如果任务非完成状态（成功或失败），则忽略
	if taskLog.Status != constant.TaskStatusSuccess && taskLog.Status != constant.TaskStatusFailed {
		return
	}

	// 查出所有启用状态的工作流
	var workflows []models.Workflow
	if err := database.DB.Where("enabled = ?", true).Find(&workflows).Error; err != nil {
		logger.Errorf("[Workflow] 查找可用工作流失败: %v", err)
		return
	}

	for _, wf := range workflows {
		if wf.FlowData == "" {
			continue
		}

		var flowData FlowData
		if err := json.Unmarshal([]byte(wf.FlowData), &flowData); err != nil {
			logger.Warnf("[Workflow] 解析工作流 %d 数据失败: %v", wf.ID, err)
			continue
		}

		// 建立 NodeID 和 TaskID 的映射
		nodeIdToTaskId := make(map[string]int)
		for _, n := range flowData.Nodes {
			if n.Data.TaskID > 0 {
				nodeIdToTaskId[n.ID] = n.Data.TaskID
			}
		}

		// 遍历工作流连线寻找目标
		for _, edge := range flowData.Edges {
			sourceTaskId := nodeIdToTaskId[edge.Source]
			if sourceTaskId != int(taskLog.TaskID) {
				continue
			}

			// 尝试多维度提取连线的条件设定
			condition := edge.SourceHandle
			if condition == "" && edge.Data.Condition != "" {
				condition = edge.Data.Condition
			}
			if condition == "" && edge.Label != "" {
				condition = edge.Label
			}

			match := false
			// 成功分支
			if (condition == "success" || condition == "on_success") && taskLog.Status == constant.TaskStatusSuccess {
				match = true
			} else if (condition == "error" || condition == "failed" || condition == "on_error") && taskLog.Status == constant.TaskStatusFailed { // 失败分支
				match = true
			} else if condition == "" || condition == "always" { // 无条件约束，总是触发
				match = true
			}

			if match {
				targetTaskId := nodeIdToTaskId[edge.Target]
				if targetTaskId > 0 {
					logger.Infof("[Workflow] 任务 #%d 执行%s，触发后续工作流 (WF: #%d) 任务 #%d", taskLog.TaskID, taskLog.Status, wf.ID, targetTaskId)
					
					// 为了缓冲并发写入和让前置任务日志落库完毕，挂载协程延迟触发下游
					go func(tid int, wfid uint) {
						time.Sleep(time.Second) 
						// 后续任务环境变量植入 Workflow 上下文
						es.ExecuteTask(tid, []string{fmt.Sprintf("WEBHOOK_WORKFLOW_ID=%d", wfid)})
					}(targetTaskId, wf.ID)
				}
			}
		}
	}
}
