package controllers

import (
	"strconv"

	"baihu/internal/services/tasks"
	"baihu/internal/utils"

	"github.com/gin-gonic/gin"
)

type ExecutorController struct {
	executorService *tasks.ExecutorService
}

func NewExecutorController(executorService *tasks.ExecutorService) *ExecutorController {
	return &ExecutorController{executorService: executorService}
}

func (ec *ExecutorController) ExecuteTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "无效的任务ID")
		return
	}

	result := ec.executorService.ExecuteTask(id)
	utils.Success(c, result)
}

func (ec *ExecutorController) ExecuteCommand(c *gin.Context) {
	var req struct {
		Command string `json:"command" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	result := ec.executorService.ExecuteCommand(req.Command)
	utils.Success(c, result)
}

func (ec *ExecutorController) GetLastResults(c *gin.Context) {
	count := 10
	if c.Query("count") != "" {
		if parsedCount, err := strconv.Atoi(c.Query("count")); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	results := ec.executorService.GetLastResults(count)
	utils.Success(c, results)
}
