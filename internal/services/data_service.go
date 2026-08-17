package services

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/relation"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

type DataService struct{}

func NewDataService() *DataService {
	return &DataService{}
}

// ExportBusinessData 智能解析依赖并导出业务数据
func (s *DataService) ExportBusinessData(taskIDs []string, envIDs []string) *models.ExportData {
	export := models.NewExportData()

	export.Tasks = s.collectTasksAndRelations(taskIDs)
	export.Envs = s.collectEnvironmentVariables(export.Tasks, envIDs)
	export.Bindings = s.collectNotifyBindings(export.Tasks)
	export.Tags = s.collectTagStorages(export.Tasks)

	return export
}

// collectTasksAndRelations 收集任务及其子任务，并加载关联关系 (Envs, Tags)
func (s *DataService) collectTasksAndRelations(taskIDs []string) []models.Task {
	if len(taskIDs) == 0 {
		return nil
	}

	var tasks []models.Task
	targetTaskIDs := make(map[string]bool)
	for _, id := range taskIDs {
		targetTaskIDs[id] = true
	}

	var initialTasks []models.Task
	database.DB.Where("id IN ?", taskIDs).Find(&initialTasks)
	tasks = append(tasks, initialTasks...)

	// 检查哪些是仓库任务，并递归查询其子任务
	var parentIDs []string
	for _, t := range initialTasks {
		if t.Type == "repo" {
			parentIDs = append(parentIDs, t.ID)
		}
	}

	if len(parentIDs) > 0 {
		var childTasks []models.Task
		database.DB.Where("repo_task_id IN ?", parentIDs).Find(&childTasks)
		for _, ct := range childTasks {
			if !targetTaskIDs[ct.ID] {
				tasks = append(tasks, ct)
				targetTaskIDs[ct.ID] = true
			}
		}
	}

	// 填充任务的 Envs 与 Tags 关系数据，以保证依赖解析和后续导入成功
	if len(tasks) > 0 {
		var exportTaskIDs []string
		for _, t := range tasks {
			exportTaskIDs = append(exportTaskIDs, t.ID)
		}

		envsMap := relation.DataRelation.LoadRelations(exportTaskIDs, constant.RelationTypeTaskEnv)
		tagsMap := relation.DataRelation.LoadTags(exportTaskIDs, constant.RelationTypeTaskTag)

		for i, t := range tasks {
			if envs, ok := envsMap[t.ID]; ok {
				tasks[i].Envs = models.BigText(strings.Join(envs, ","))
			}
			if tags, ok := tagsMap[t.ID]; ok {
				tasks[i].Tags = strings.Join(tags, ",")
			}
		}
	}

	return tasks
}

// collectEnvironmentVariables 收集所需和任务所依赖的环境变量
func (s *DataService) collectEnvironmentVariables(tasks []models.Task, envIDs []string) []models.EnvironmentVariable {
	targetEnvIDs := make(map[string]bool)
	for _, id := range envIDs {
		targetEnvIDs[id] = true
	}

	// 从任务中解析依赖的环境变量
	for _, t := range tasks {
		if t.Envs != "" {
			envArray := strings.Split(string(t.Envs), ",")
			for _, eID := range envArray {
				eID = strings.TrimSpace(eID)
				if eID != "" {
					targetEnvIDs[eID] = true
				}
			}
		}
	}

	if len(targetEnvIDs) == 0 {
		return nil
	}

	var finalEnvIDs []string
	for id := range targetEnvIDs {
		finalEnvIDs = append(finalEnvIDs, id)
	}

	var envs []models.EnvironmentVariable
	database.DB.Where("id IN ?", finalEnvIDs).Find(&envs)
	return envs
}

// collectNotifyBindings 收集相关的通知规则 (NotifyBindings)
func (s *DataService) collectNotifyBindings(tasks []models.Task) []models.NotifyBinding {
	if len(tasks) == 0 {
		return nil
	}

	var taskIDList []string
	for _, t := range tasks {
		taskIDList = append(taskIDList, t.ID)
	}

	var bindings []models.NotifyBinding
	database.DB.Where("type = ? AND data_id IN ?", "task", taskIDList).Find(&bindings)
	return bindings
}

// collectTagStorages 收集标签定义 (DataStorage)
func (s *DataService) collectTagStorages(tasks []models.Task) []models.DataStorage {
	targetTagNames := make(map[string]bool)
	for _, t := range tasks {
		if t.Tags != "" {
			tagArray := strings.Split(t.Tags, ",")
			for _, tagName := range tagArray {
				tagName = strings.TrimSpace(tagName)
				if tagName != "" {
					targetTagNames[tagName] = true
				}
			}
		}
	}

	if len(targetTagNames) == 0 {
		return nil
	}

	var finalTagNames []string
	for name := range targetTagNames {
		finalTagNames = append(finalTagNames, name)
	}

	var tagStorages []models.DataStorage
	database.DB.Where("type = ? AND name IN ?", constant.RelationTypeTaskTag, finalTagNames).Find(&tagStorages)
	return tagStorages
}

// ImportBusinessData 导入业务数据
func (s *DataService) ImportBusinessData(data *models.ExportData) error {
	tx := database.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	var adminUser models.User
	if err := tx.Where("role = ?", constant.AdminRole).First(&adminUser).Error; err != nil {
		tx.Rollback()
		return err
	}

	importer := &businessImporter{
		tx:      tx,
		adminID: adminUser.ID,
	}

	// 1. 导入环境变量
	if err := importer.importEnvs(data.Envs); err != nil {
		tx.Rollback()
		return err
	}

	// 2. 导入标签定义 (DataStorage)
	if err := importer.importTags(data.Tags); err != nil {
		tx.Rollback()
		return err
	}

	// 3. 导入任务及关联映射关系
	if err := importer.importTasks(data.Tasks); err != nil {
		tx.Rollback()
		return err
	}

	// 4. 导入通知规则
	if err := importer.importBindings(data.Bindings); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

type businessImporter struct {
	tx      *gorm.DB
	adminID string
}

func (importer *businessImporter) importEnvs(envs []models.EnvironmentVariable) error {
	if len(envs) == 0 {
		return nil
	}
	for _, env := range envs {
		env.UserID = importer.adminID
		if err := importer.tx.Save(&env).Error; err != nil {
			return err
		}
	}
	return nil
}

func (importer *businessImporter) importTags(tags []models.DataStorage) error {
	if len(tags) == 0 {
		return nil
	}
	for _, tagStorage := range tags {
		if err := importer.tx.Save(&tagStorage).Error; err != nil {
			return err
		}
	}
	return nil
}

func (importer *businessImporter) importTasks(tasks []models.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	for _, task := range tasks {
		if err := importer.tx.Save(&task).Error; err != nil {
			return err
		}

		if err := importer.importTaskRelations(task); err != nil {
			return err
		}
	}
	return nil
}

func (importer *businessImporter) importTaskRelations(task models.Task) error {
	if err := importer.importTaskTagRelations(task); err != nil {
		return err
	}
	return importer.importTaskEnvRelations(task)
}

func (importer *businessImporter) importTaskTagRelations(task models.Task) error {
	if task.Tags == "" {
		return nil
	}
	if err := importer.tx.Where("data_id = ? AND type = ?", task.ID, constant.RelationTypeTaskTag).Delete(&models.DataRelation{}).Error; err != nil {
		return err
	}
	tags := strings.Split(task.Tags, ",")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		var storage models.DataStorage
		res := importer.tx.Where("type = ? AND name = ?", constant.RelationTypeTaskTag, tag).Limit(1).Find(&storage)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			storage = models.DataStorage{
				ID:        xid.New().String(),
				Type:      constant.RelationTypeTaskTag,
				Name:      tag,
				CreatedAt: models.Now(),
				UpdatedAt: models.Now(),
			}
			if err := importer.tx.Create(&storage).Error; err != nil {
				return err
			}
		}
		rel := models.DataRelation{
			ID:        xid.New().String(),
			DataID:    task.ID,
			RelateID:  storage.ID,
			Type:      constant.RelationTypeTaskTag,
			CreatedAt: models.Now(),
			UpdatedAt: models.Now(),
		}
		if err := importer.tx.Create(&rel).Error; err != nil {
			return err
		}
	}
	return nil
}

func (importer *businessImporter) importTaskEnvRelations(task models.Task) error {
	if string(task.Envs) == "" {
		return nil
	}
	if err := importer.tx.Where("data_id = ? AND type = ?", task.ID, constant.RelationTypeTaskEnv).Delete(&models.DataRelation{}).Error; err != nil {
		return err
	}
	ids := strings.Split(string(task.Envs), ",")
	for _, relateID := range ids {
		relateID = strings.TrimSpace(relateID)
		if relateID == "" {
			continue
		}
		rel := models.DataRelation{
			ID:        xid.New().String(),
			DataID:    task.ID,
			RelateID:  relateID,
			Type:      constant.RelationTypeTaskEnv,
			CreatedAt: models.Now(),
			UpdatedAt: models.Now(),
		}
		if err := importer.tx.Create(&rel).Error; err != nil {
			return err
		}
	}
	return nil
}

func (importer *businessImporter) importBindings(bindings []models.NotifyBinding) error {
	if len(bindings) == 0 {
		return nil
	}
	for _, binding := range bindings {
		if err := importer.tx.Save(&binding).Error; err != nil {
			return err
		}
	}
	return nil
}
