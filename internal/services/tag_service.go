package services

import (
	"errors"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

type TagService struct{}

func NewTagService() *TagService {
	return &TagService{}
}

// TagWithCount 带相关联计数的标签数据
type TagWithCount struct {
	models.DataStorage
	AssociationCount int64 `json:"association_count"`
}

// GetTagsWithPagination 分页获取标签列表
func (s *TagService) GetTagsWithPagination(page, pageSize int, name string, relType string) ([]TagWithCount, int64, error) {
	var total int64
	db := database.DB.Model(&models.DataStorage{})
	if relType != "" {
		db = db.Where("type = ?", relType)
	}
	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}

	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var storages []models.DataStorage
	offset := (page - 1) * pageSize
	err = db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&storages).Error
	if err != nil {
		return nil, 0, err
	}

	var results []TagWithCount
	for _, storage := range storages {
		var count int64
		database.DB.Model(&models.DataRelation{}).Where("relate_id = ? AND type = ?", storage.ID, storage.Type).Count(&count)
		results = append(results, TagWithCount{
			DataStorage:      storage,
			AssociationCount: count,
		})
	}

	return results, total, nil
}

// CreateTag 手动创建标签
func (s *TagService) CreateTag(name string, relType string) (*models.DataStorage, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("标签名称不能为空")
	}
	var count int64
	err := database.DB.Model(&models.DataStorage{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("标签名称已存在")
	}

	storage := models.DataStorage{
		ID:        xid.New().String(),
		Type:      relType,
		Name:      name,
		CreatedAt: models.Now(),
		UpdatedAt: models.Now(),
	}
	if err := database.DB.Create(&storage).Error; err != nil {
		return nil, err
	}
	return &storage, nil
}

// RenameTag 重命名标签
func (s *TagService) RenameTag(id string, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return errors.New("标签名称不能为空")
	}
	var storage models.DataStorage
	if err := database.DB.First(&storage, "id = ?", id).Error; err != nil {
		return err
	}

	var count int64
	err := database.DB.Model(&models.DataStorage{}).Where("name = ? AND id != ?", newName, id).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("标签名称已存在")
	}

	return database.DB.Model(&storage).Updates(map[string]interface{}{
		"name":       newName,
		"updated_at": models.Now(),
	}).Error
}

// DeleteTag 删除标签并清理关联
func (s *TagService) DeleteTag(id string) error {
	var storage models.DataStorage
	if err := database.DB.First(&storage, "id = ?", id).Error; err != nil {
		return err
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&storage).Error; err != nil {
			return err
		}
		return tx.Where("relate_id = ? AND type = ?", id, storage.Type).Delete(&models.DataRelation{}).Error
	})
}

// TagResourceItem 标签关联的资源详细项
type TagResourceItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`      // "task", "repo", "env"
	TypeName string `json:"type_name"` // "定时任务", "仓库同步", "环境变量"
	Remark   string `json:"remark"`
	Extra    string `json:"extra"`    // 命令或环境变量值预览
	Schedule string `json:"schedule"` // 针对任务的 cron 表达式
	Enabled  bool   `json:"enabled"`
}

// TagResourcesResponse 标签关联资源响应结构
type TagResourcesResponse struct {
	TagID     string            `json:"tag_id"`
	TagName   string            `json:"tag_name"`
	TagType   string            `json:"tag_type"`
	Total     int               `json:"total"`
	Resources []TagResourceItem `json:"resources"`
}

// GetTagResources 获取指定标签关联的实际资源列表（任务、仓库、环境变量）
func (s *TagService) GetTagResources(id string) (*TagResourcesResponse, error) {
	var storage models.DataStorage
	if err := database.DB.First(&storage, "id = ?", id).Error; err != nil {
		return nil, errors.New("标签不存在")
	}

	resp := &TagResourcesResponse{
		TagID:     storage.ID,
		TagName:   storage.Name,
		TagType:   storage.Type,
		Resources: make([]TagResourceItem, 0),
	}

	var relations []models.DataRelation
	err := database.DB.Where("relate_id = ? AND type = ?", id, storage.Type).Find(&relations).Error
	if err != nil {
		return nil, err
	}

	if len(relations) == 0 {
		return resp, nil
	}

	dataIDs := make([]string, 0, len(relations))
	for _, rel := range relations {
		dataIDs = append(dataIDs, rel.DataID)
	}

	if storage.Type == constant.RelationTypeTaskTag {
		var tasks []models.Task
		if err := database.DB.Where("id IN ?", dataIDs).Order("name asc").Find(&tasks).Error; err != nil {
			return nil, err
		}
		for _, task := range tasks {
			typeName := "定时任务"
			if task.Type == constant.TaskTypeRepo {
				typeName = "仓库同步"
			}
			enabled := true
			if task.Enabled != nil {
				enabled = *task.Enabled
			}
			resp.Resources = append(resp.Resources, TagResourceItem{
				ID:       task.ID,
				Name:     task.Name,
				Type:     task.Type,
				TypeName: typeName,
				Remark:   task.Remark,
				Extra:    string(task.Command),
				Schedule: task.Schedule,
				Enabled:  enabled,
			})
		}
	} else if storage.Type == constant.RelationTypeEnvTag {
		var envs []models.EnvironmentVariable
		if err := database.DB.Where("id IN ?", dataIDs).Order("name asc").Find(&envs).Error; err != nil {
			return nil, err
		}
		for _, env := range envs {
			enabled := true
			if env.Enabled != nil {
				enabled = *env.Enabled
			}
			resp.Resources = append(resp.Resources, TagResourceItem{
				ID:       env.ID,
				Name:     env.Name,
				Type:     "env",
				TypeName: "环境变量",
				Remark:   env.Remark,
				Extra:    string(env.Value),
				Enabled:  enabled,
			})
		}
	}

	resp.Total = len(resp.Resources)
	return resp, nil
}
