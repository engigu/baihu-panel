package common

import (
	"encoding/json"
	"strings"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils/idgen"
	"gorm.io/gorm"
)

type RelationService struct{}

func NewRelationService() *RelationService {
	return &RelationService{}
}

// UpdateRelations 更新关联关系（先删除旧的，再建立新的）
func (s *RelationService) UpdateRelations(dataID, relationType, storageType string, keys []string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 删除旧的关联关系
		if err := tx.Where("data_id = ? AND type = ?", dataID, relationType).Delete(&models.DataRelation{}).Error; err != nil {
			return err
		}

		// 2. 建立新的关联关系
		for i, key := range keys {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}

			// 查找或创建 DataStorage
			var storage models.DataStorage
			res := tx.Where("type = ? AND key = ?", storageType, key).Limit(1).Find(&storage)
			if res.RowsAffected == 0 {
				dataJson, _ := json.Marshal(map[string]string{"name": key})
				storage = models.DataStorage{
					ID:   idgen.GenerateID(),
					Type: storageType,
					Key:  key,
					Data: models.BigText(dataJson),
				}
				if err := tx.Create(&storage).Error; err != nil {
					return err
				}
			}

			// 创建 DataRelation
			relation := models.DataRelation{
				ID:        idgen.GenerateID(),
				DataID:    dataID,
				RelateID:  storage.ID,
				Type:      relationType,
				SortOrder: i,
			}
			if err := tx.Create(&relation).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetRelatedKeys 获取关联的所有 Key
func (s *RelationService) GetRelatedKeys(dataID, relationType string) ([]string, error) {
	var keys []string
	err := database.DB.Table(models.DataRelation{}.TableName()).
		Select("ds.key").
		Joins("JOIN " + models.DataStorage{}.TableName() + " ds ON " + models.DataRelation{}.TableName() + ".relate_id = ds.id").
		Where(models.DataRelation{}.TableName()+".data_id = ? AND "+models.DataRelation{}.TableName()+".type = ?", dataID, relationType).
		Order(models.DataRelation{}.TableName() + ".sort_order ASC").
		Scan(&keys).Error
	return keys, err
}

// GetDistinctKeys 获取指定类型的所有唯一 Key
func (s *RelationService) GetDistinctKeys(storageType string) ([]string, error) {
	var keys []string
	err := database.DB.Model(&models.DataStorage{}).
		Where("type = ?", storageType).
		Pluck("key", &keys).Error
	return keys, err
}
