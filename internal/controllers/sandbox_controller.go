package controllers

import (
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

type SandboxController struct {
	sandboxService *services.SandboxService
}

func NewSandboxController(sandboxService *services.SandboxService) *SandboxController {
	return &SandboxController{
		sandboxService: sandboxService,
	}
}

// 注意: 使用标准的 gin.Context，因为之前是 gin.Context 的写法
func (sc *SandboxController) GetSandboxProfiles(c *gin.Context) {
	list, err := sc.sandboxService.GetSandboxProfiles()
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.Success(c, list)
}

func (sc *SandboxController) GetSandboxProfileByID(c *gin.Context) {
	id := c.Param("id")
	profile, err := sc.sandboxService.GetSandboxProfileByID(id)
	if err != nil {
		utils.NotFound(c, "沙箱配置不存在")
		return
	}
	utils.Success(c, profile)
}

func (sc *SandboxController) CreateSandboxProfile(c *gin.Context) {
	var req models.SandboxProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	if req.Name == "" {
		utils.BadRequest(c, "沙箱名称不能为空")
		return
	}
	err := sc.sandboxService.CreateSandboxProfile(&req)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.Success(c, req)
}

func (sc *SandboxController) UpdateSandboxProfile(c *gin.Context) {
	id := c.Param("id")
	var req models.SandboxProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	if req.Name == "" {
		utils.BadRequest(c, "沙箱名称不能为空")
		return
	}
	profile, err := sc.sandboxService.UpdateSandboxProfile(id, &req)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.Success(c, profile)
}

func (sc *SandboxController) DeleteSandboxProfile(c *gin.Context) {
	id := c.Param("id")
	err := sc.sandboxService.DeleteSandboxProfile(id)
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.Success(c, true)
}

func (sc *SandboxController) RepairSandboxDirectories(c *gin.Context) {
	sc.sandboxService.InitSandboxDirectories()
	utils.SuccessMsg(c, "修复并重新生成所有沙箱目录成功")
}
