package controllers

import (
	"strconv"

	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type ExecutorController struct {
	executorService *tasks.ExecutorService
}

func NewExecutorController(executorService *tasks.ExecutorService) *ExecutorController {
	return &ExecutorController{executorService: executorService}
}

// ExecuteTask 运行任务
// @Summary 运行任务
// @Description 立即执行指定的任务
// @Tags 任务执行
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Param body body object false "执行参数 (envs: 环境变量字典)"
// @Success 200 {object} utils.Response{data=vo.ExecutionResultVO}
// @Failure 400 {object} utils.Response
// @Router /execute/task/{id} [post]
func (ec *ExecutorController) ExecuteTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的任务ID")
		return
	}

	var req struct {
		Envs map[string]string `json:"envs"`
	}
	// 尝试绑定 JSON 体，但不强制要求
	_ = c.ShouldBindJSON(&req)

	var extraEnvs []string
	if req.Envs != nil {
		for k, v := range req.Envs {
			extraEnvs = append(extraEnvs, k+"="+v)
		}
	}

	result := ec.executorService.ExecuteTask(id, extraEnvs)
	utils.Success(c, vo.ToExecutionResultVO(result))
}

// ExecuteCommand 执行命令
func (ec *ExecutorController) ExecuteCommand(c *gin.Context) {
	var req struct {
		Command string `json:"command" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result := ec.executorService.ExecuteCommand(req.Command)
	utils.Success(c, vo.ToExecutionResultVO(result))
}

// GetLastResults 获取最新执行结果
// @Summary 获取最新执行结果
// @Description 获取最新任务或命令执行的结果列表
// @Tags 任务执行
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param count query int false "数量 (默认 10)"
// @Success 200 {object} utils.Response{data=[]vo.ExecutionResultVO}
// @Router /execute/results [get]
func (ec *ExecutorController) GetLastResults(c *gin.Context) {
	count := 10
	if c.Query("count") != "" {
		if parsedCount, err := strconv.Atoi(c.Query("count")); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	results := ec.executorService.GetLastResults(count)
	utils.Success(c, vo.ToExecutionResultVOList(results))
}
