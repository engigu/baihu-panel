package services

import (
	"errors"
	"strings"

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
