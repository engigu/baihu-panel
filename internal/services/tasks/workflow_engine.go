package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
)

type FlowNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		TaskID      int    `json:"taskId"` // Store the Baihu Task ID
		NodeType    string `json:"nodeType"`
		ControlType string `json:"controlType"`
		Config      string `json:"config"` // JSON string for specific control configuration
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
		NodeType  string `json:"nodeType"`
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

	// 取出此任务记录的 WorkflowID 和 WorkflowRunID
	// 如果不是由工作流发起的任务，则无需向下游传递
	if taskLog.WorkflowID == nil || *taskLog.WorkflowID == "" || taskLog.WorkflowRunID == "" {
		// 仍然需要支持由于普通任务完成意外触发了某个包含该任务的WF（兼容老逻辑）
		// 但为了严格的运行追踪，我们最好只处理显式关联的流水线。
		// 这里暂不 return，允许未标记的任务也触发对应连线，并为其生成一条新的 run_id
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
			logger.Warnf("[Workflow] 解析工作流 %s 数据失败: %v", wf.ID, err)
			continue
		}

		// 提取任务输出中的变量 (BAIHU_OUT_KEY=VALUE)
		capturedEnvs := make([]string, 0)
		if taskLog.Output != "" {
			rawOutput, _ := utils.DecompressFromBase64(taskLog.Output)
			lines := strings.Split(rawOutput, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "BAIHU_OUT_") {
					// 将 BAIHU_OUT_ 转换为下游可用的环境变量（去掉 OUT_ 标识或保留原样均可，通常去掉更直观）
					// 这里保留 BAIHU_ 标识，但去掉 OUT_，例如 BAIHU_OUT_FILE=x -> BAIHU_FILE=x
					kv := strings.TrimPrefix(line, "BAIHU_OUT_")
					capturedEnvs = append(capturedEnvs, "BAIHU_"+kv)
				}
			}
		}

		// 建立 NodeID 到 TaskID 和 NodeType 的映射
		nodeIdToTaskId := make(map[string]int)
		nodeIdToType := make(map[string]string)
		nodeIdToConfig := make(map[string]string)
		for _, n := range flowData.Nodes {
			if n.Data.TaskID != 0 {
				nodeIdToTaskId[n.ID] = n.Data.TaskID
				nodeIdToType[n.ID] = n.Data.NodeType
				nodeIdToConfig[n.ID] = n.Data.Config
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
			if (condition == constant.WorkflowConditionSuccess || condition == constant.WorkflowConditionOnSuccess) && taskLog.Status == constant.TaskStatusSuccess {
				match = true
			} else if (condition == constant.WorkflowConditionError || condition == constant.WorkflowConditionFailed || condition == constant.WorkflowConditionOnError) && taskLog.Status == constant.TaskStatusFailed { // 失败分支
				match = true
			} else if condition == "" || condition == constant.WorkflowConditionAlways { // 无条件约束，总是触发
				match = true
			}

			if match {
				targetTaskId := nodeIdToTaskId[edge.Target]
				if targetTaskId > 0 {
					logger.Infof("[Workflow] 任务 #%d 执行%s，触发后续工作流 (WF: #%s) 任务 #%d", taskLog.TaskID, taskLog.Status, wf.ID, targetTaskId)
					// 如果当前任务属于某 Run，则延续；否则开启新的 Run 追踪支线
					runID := taskLog.WorkflowRunID
					if runID == "" {
						runID = fmt.Sprintf("WF-RUN-%d", time.Now().UnixMilli())
					}
					
					// 为了缓冲并发写入和让前置任务日志落库完毕，挂载协程延迟触发下游
					go func(tid int, wfid string, rid string, extraEnvs []string, nType string, nConfig string) {
						// 1. 如果是控制节点，由控制逻辑处理
						if nType == constant.TaskTypeControl {
							es.triggerControlNode(tid, nConfig, wfid, rid, extraEnvs)
							return
						}

						// 2. 普通任务节点，正常延迟 1s 触发执行
						time.Sleep(time.Second) 
						// 后续任务环境变量植入 Workflow 上下文
						envs := []string{
							fmt.Sprintf("BAIHU_WF_ID=%s", wfid),
							fmt.Sprintf("BAIHU_WF_RUN_ID=%s", rid),
						}
						envs = append(envs, extraEnvs...)
						es.ExecuteTask(tid, envs)
					}(targetTaskId, wf.ID, runID, capturedEnvs, nodeIdToType[edge.Target], nodeIdToConfig[edge.Target])
				}
			}
		}
	}
}

// triggerControlNode 处理虚拟控制节点（如：延时、分支判断、WebHook）
func (es *ExecutorService) triggerControlNode(targetNodeTaskId int, config string, wfID string, wfRunID string, envs []string) {
	// 获取该节点的详细配置
	var task models.Task
	if err := database.DB.First(&task, targetNodeTaskId).Error; err != nil {
		logger.Errorf("[Workflow] 控制节点任务 #%d 获取失败: %v", targetNodeTaskId, err)
		return
	}

	// 创建一个特殊的控制节点执行日志
	logService := &TaskLogService{}
	taskLog, _ := logService.CreateEmptyLog(uint(targetNodeTaskId), "Workflow Control: "+task.Name, &wfID, wfRunID)

	// 根据子类型处理逻辑
	// 注意：前端拖拽生成的控制节点 TaskID 我们定死为 -1，内部根据 ControlType/Tags 识别
	nodeType := task.Tags // 延时节点对应 Tags="delay"
	
	switch nodeType {
	case "delay":
		// 读取延时时间（秒）
		delaySec := 5 // 默认 5s
		if config != "" {
			fmt.Sscanf(config, "%d", &delaySec)
		}
		logger.Infof("[Workflow] 控制节点 %s #%d (延时) 开始等待 %d 秒 (Run: %s)", task.Name, targetNodeTaskId, delaySec, wfRunID)
		
		go func() {
			time.Sleep(time.Duration(delaySec) * time.Second)
			// 完成虚拟任务
			taskLog.Status = constant.TaskStatusSuccess
			now := models.Now()
			taskLog.EndTime = &now
			taskLog.Output, _ = utils.CompressToBase64(fmt.Sprintf("Wait completed after %d seconds.", delaySec))
			logService.ProcessTaskCompletion(taskLog)

			// 触发下一级
			es.TriggerWorkflowNextTasks(taskLog)
		}()
	default:
		// 未知类型直接设为成功并跳过
		taskLog.Status = constant.TaskStatusSuccess
		now := models.Now()
		taskLog.EndTime = &now
		logService.ProcessTaskCompletion(taskLog)
		es.TriggerWorkflowNextTasks(taskLog)
	}
}

// TriggerWorkflow 手动或定时根据 ID 立即执行工作流（找出根节点触发）
func (es *ExecutorService) TriggerWorkflow(workflowID string, extraEnvs []string) error {
	var wf models.Workflow
	if err := database.DB.Where("id = ?", workflowID).First(&wf).Error; err != nil {
		return fmt.Errorf("工作流未找到: %v", err)
	}

	if wf.FlowData == "" || !wf.Enabled {
		return fmt.Errorf("工作流为空或已被禁用")
	}

	var flowData FlowData
	if err := json.Unmarshal([]byte(wf.FlowData), &flowData); err != nil {
		return fmt.Errorf("解析工作流数据失败: %v", err)
	}

	// 找出所有存在入度的 NodeID
	hasIncomingEdges := make(map[string]bool)
	for _, edge := range flowData.Edges {
		hasIncomingEdges[edge.Target] = true
	}

	// 收集所有根节点 TaskID
	var rootTaskIDs []int
	for _, node := range flowData.Nodes {
		if !hasIncomingEdges[node.ID] && node.Data.TaskID > 0 {
			rootTaskIDs = append(rootTaskIDs, node.Data.TaskID)
		}
	}

	if len(rootTaskIDs) == 0 {
		return fmt.Errorf("工作流中没有找到可作为起点的独立根节点任务")
	}

	// 更新最后运行时间
	now := models.LocalTime(time.Now())
	database.DB.Model(&wf).Update("last_run", &now)

	// 并发触发这些根节点
	runID := fmt.Sprintf("WF-RUN-%d", time.Now().UnixMilli())
	for _, node := range flowData.Nodes {
		if hasIncomingEdges[node.ID] {
			continue
		}
		
		tid := node.Data.TaskID
		if tid == 0 {
			continue
		}

		go func(taskID int, nType string, nConfig string) {
			// 如果起点就是个控制节点（比如延时启动 - 虽然少见）
			if nType == constant.TaskTypeControl {
				es.triggerControlNode(taskID, nConfig, wf.ID, runID, extraEnvs)
				return
			}

			logger.Infof("[Workflow] 触发启动工作流 (WF: #%s), 启动根节点任务 #%d", wf.ID, taskID)
			envs := []string{
				fmt.Sprintf("BAIHU_WF_ID=%s", wf.ID),
				fmt.Sprintf("BAIHU_WF_RUN_ID=%s", runID),
				fmt.Sprintf("BAIHU_WF_TRIGGER=manual"),
			}
			envs = append(envs, extraEnvs...)
			es.ExecuteTask(taskID, envs)
		}(tid, node.Data.NodeType, node.Data.Config)
	}

	return nil
}
