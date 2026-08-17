package relation

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/rs/xid"
)

type DataRelationService struct{}

var DataRelation = &DataRelationService{}

// SaveTags 保存带有 Storage (文本标签) 的关系映射
func (s *DataRelationService) SaveTags(dataID string, relType string, tagsStr string) {
	database.DB.Where("data_id = ? AND type = ?", dataID, relType).Delete(&models.DataRelation{})
	if tagsStr == "" {
		return
	}
	tags := strings.Split(tagsStr, ",")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		var storage models.DataStorage
		res := database.DB.Where("type = ? AND name = ?", relType, tag).Limit(1).Find(&storage)
		if res.RowsAffected == 0 {
			storage = models.DataStorage{
				ID:        xid.New().String(),
				Type:      relType,
				Name:      tag,
				CreatedAt: models.Now(),
				UpdatedAt: models.Now(),
			}
			database.DB.Create(&storage)
		}
		relation := models.DataRelation{
			ID:        xid.New().String(),
			DataID:    dataID,
			RelateID:  storage.ID,
			Type:      relType,
			CreatedAt: models.Now(),
			UpdatedAt: models.Now(),
		}
		database.DB.Create(&relation)
	}
}

// LoadTags 加载带有 Storage (文本标签) 的映射，返回 map[DataID][]TagName
func (s *DataRelationService) LoadTags(dataIDs []string, relType string) map[string][]string {
	if len(dataIDs) == 0 {
		return nil
	}
	var relations []models.DataRelation
	database.DB.Where("data_id IN ? AND type = ?", dataIDs, relType).Find(&relations)

	if len(relations) == 0 {
		return nil
	}

	var relateIDs []string
	for _, r := range relations {
		relateIDs = append(relateIDs, r.RelateID)
	}

	var storages []models.DataStorage
	database.DB.Where("id IN ?", relateIDs).Find(&storages)

	storageMap := make(map[string]string)
	for _, storage := range storages {
		storageMap[storage.ID] = storage.Name
	}

	resultMap := make(map[string][]string)
	for _, r := range relations {
		if name, ok := storageMap[r.RelateID]; ok {
			resultMap[r.DataID] = append(resultMap[r.DataID], name)
		}
	}
	return resultMap
}

// SaveRelations 保存单纯的关系映射 (例如 ID关联)
func (s *DataRelationService) SaveRelations(dataID string, relType string, relateIDsStr string) {
	database.DB.Where("data_id = ? AND type = ?", dataID, relType).Delete(&models.DataRelation{})
	if relateIDsStr == "" {
		return
	}
	ids := strings.Split(relateIDsStr, ",")
	for _, relateID := range ids {
		relateID = strings.TrimSpace(relateID)
		if relateID == "" {
			continue
		}
		relation := models.DataRelation{
			ID:        xid.New().String(),
			DataID:    dataID,
			RelateID:  relateID,
			Type:      relType,
			CreatedAt: models.Now(),
			UpdatedAt: models.Now(),
		}
		database.DB.Create(&relation)
	}
}

// LoadRelations 加载单纯的关系映射，返回 map[DataID][]RelateID
func (s *DataRelationService) LoadRelations(dataIDs []string, relType string) map[string][]string {
	if len(dataIDs) == 0 {
		return nil
	}
	var relations []models.DataRelation
	database.DB.Where("data_id IN ? AND type = ?", dataIDs, relType).Find(&relations)

	resultMap := make(map[string][]string)
	for _, r := range relations {
		resultMap[r.DataID] = append(resultMap[r.DataID], r.RelateID)
	}
	return resultMap
}

// CleanRelations 删除某种类型的所有关联映射
func (s *DataRelationService) CleanRelations(dataID string, relType string) {
	database.DB.Where("data_id = ? AND type = ?", dataID, relType).Delete(&models.DataRelation{})
}

// GetAllTags 获取全局范围内某种类型的所有的 Tag Name
func (s *DataRelationService) GetAllTags(relType string) ([]string, error) {
	var storages []models.DataStorage
	err := database.DB.Where("type = ?", relType).Find(&storages).Error
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, s := range storages {
		tags = append(tags, s.Name)
	}
	return tags, nil
}
