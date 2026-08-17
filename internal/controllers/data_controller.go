package controllers

import (
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

type DataController struct {
	dataService     *services.DataService
	taskController  *TaskController
	envController   *EnvController
}

func NewDataController(tc *TaskController, ec *EnvController) *DataController {
	return &DataController{
		dataService:    services.NewDataService(),
		taskController: tc,
		envController:  ec,
	}
}

// ExportBusinessData 导出业务数据
func (dc *DataController) ExportBusinessData(c *gin.Context) {
	var req struct {
		TaskIDs []string `json:"task_ids"`
		EnvIDs  []string `json:"env_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	exportData := dc.dataService.ExportBusinessData(req.TaskIDs, req.EnvIDs)
	utils.Success(c, exportData)
}

// ImportBusinessData 导入业务数据
func (dc *DataController) ImportBusinessData(c *gin.Context) {
	var req models.ExportData
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if req.Version == "" {
		utils.BadRequest(c, "无效的导入数据格式")
		return
	}

	// 停止相关的定时任务
	if len(req.Tasks) > 0 {
		for _, task := range req.Tasks {
			dc.taskController.executorService.RemoveCronTask(task.ID)
			dc.taskController.executorService.GetScheduler().StopTask(task.ID)
		}
	}

	// 导入数据
	if err := dc.dataService.ImportBusinessData(&req); err != nil {
		utils.ServerError(c, "导入失败: "+err.Error())
		return
	}

	// 重新启动任务和通知相关的代理
	if len(req.Tasks) > 0 {
		for i := range req.Tasks {
			task := &req.Tasks[i]
			if utils.DerefBool(task.Enabled, true) && (task.AgentID == nil || *task.AgentID == "") {
				dc.taskController.executorService.AddCronTask(task)
			}
			if task.AgentID != nil && *task.AgentID != "" {
				dc.taskController.agentWSManager.BroadcastTasks(*task.AgentID)
			}
		}
	}

	utils.SuccessMsg(c, "导入成功")
}
