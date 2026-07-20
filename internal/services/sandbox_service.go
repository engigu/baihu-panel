package services

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
)

type SandboxService struct {
}

func NewSandboxService() *SandboxService {
	return &SandboxService{}
}

func (s *SandboxService) GetSandboxProfiles() ([]models.SandboxProfile, error) {
	var list []models.SandboxProfile
	err := database.DB.Order("created_at desc").Find(&list).Error
	return list, err
}

func (s *SandboxService) GetSandboxProfileByID(id string) (*models.SandboxProfile, error) {
	var profile models.SandboxProfile
	err := database.DB.Where("id = ?", id).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// ensureSandboxDir 自动在 scripts/sandbox/<uid>/ 建立目录并将其所有权赋予沙箱的 UID/GID
func (s *SandboxService) ensureSandboxDir(uid, gid int) {
	if uid <= 0 {
		return
	}
	absScriptsDir := utils.ResolveAbsScriptsDir()
	sandboxPath := filepath.Join(absScriptsDir, "sandbox", fmt.Sprintf("%d", uid))

	// 创建目录 (权限默认为 775，确保组成员与所有者可以自由读写)
	if err := os.MkdirAll(sandboxPath, 0775); err == nil {
		if runtime.GOOS != "windows" {
			// 在 Unix 平台下执行 chown 以将目录所有权交给沙箱对应的 UID/GID 用户
			_ = os.Chown(sandboxPath, uid, gid)
		}
	}
}

// InitSandboxDirectories 系统启动时初始化，确保所有在册的沙箱模板其关联的 UID 隔离目录和权限全部就位
func (s *SandboxService) InitSandboxDirectories() {
	list, err := s.GetSandboxProfiles()
	if err != nil {
		return
	}
	
	// 1. 确保基础 scripts/sandbox 根目录存在
	absScriptsDir := utils.ResolveAbsScriptsDir()
	baseSandboxPath := filepath.Join(absScriptsDir, "sandbox")
	if err := os.MkdirAll(baseSandboxPath, 0775); err == nil {
		if runtime.GOOS != "windows" {
			// 在 Unix 下，我们需要确保白虎面板进程可以拥有并往其中读写目录
			// 默认过户为当前运行用户，或保留 775 以允许多用户使用
			_ = os.Chown(baseSandboxPath, os.Getuid(), os.Getgid())
		}
	}

	// 2. 依次初始化并修复每个已配置 UID 的沙箱目录
	for _, profile := range list {
		s.ensureSandboxDir(profile.UID, profile.GID)
	}
}

func (s *SandboxService) CreateSandboxProfile(profile *models.SandboxProfile) error {
	profile.ID = utils.GenerateID()
	profile.CreatedAt = models.Now()
	profile.UpdatedAt = models.Now()
	
	err := database.DB.Create(profile).Error
	if err == nil {
		// 在数据库写入成功后自动在 scripts/sandbox/<uid>/ 创建强绑定目录
		s.ensureSandboxDir(profile.UID, profile.GID)
	}
	return err
}

func (s *SandboxService) UpdateSandboxProfile(id string, update *models.SandboxProfile) (*models.SandboxProfile, error) {
	profile, err := s.GetSandboxProfileByID(id)
	if err != nil {
		return nil, err
	}
	profile.Name = update.Name
	profile.Description = update.Description
	profile.MemoryLimit = update.MemoryLimit
	profile.NprocLimit = update.NprocLimit
	profile.UID = update.UID
	profile.GID = update.GID
	profile.HideSystemEtc = update.HideSystemEtc
	profile.UpdatedAt = models.Now()

	err = database.DB.Save(profile).Error
	if err == nil {
		// 更新配置时同步建立或修正沙箱绑定目录
		s.ensureSandboxDir(profile.UID, profile.GID)
	}
	return profile, err
}

func (s *SandboxService) DeleteSandboxProfile(id string) error {
	// 将绑定了该沙箱的 Task 的 SandboxProfileID 置为空
	database.DB.Model(&models.Task{}).Where("sandbox_profile_id = ?", id).Update("sandbox_profile_id", nil)
	return database.DB.Where("id = ?", id).Delete(&models.SandboxProfile{}).Error
}

// GetSandboxConfig 根据 ID 获取并装配沙箱配置模型（提供防守性与默认空指针处理）
func (s *SandboxService) GetSandboxConfig(id *string) *models.SandboxConfig {
	if id == nil || *id == "" || *id == "none" {
		return nil
	}
	profile, err := s.GetSandboxProfileByID(*id)
	if err != nil {
		return nil
	}
	return &models.SandboxConfig{
		UseSandbox:    true,
		MemoryLimit:   profile.MemoryLimit,
		NprocLimit:    profile.NprocLimit,
		UID:           profile.UID,
		GID:           profile.GID,
		HideSystemEtc: profile.HideSystemEtc,
	}
}
