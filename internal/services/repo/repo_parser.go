package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/relation"
	"github.com/engigu/baihu-panel/internal/utils"
)

// ParseRepoScriptsAndAddCron 扫描仓库目录中的脚本，解析 cron 和环境注释，并注册任务
func ParseRepoScriptsAndAddCron(taskID string, logWriter io.Writer, forceCommentToTask bool) ([]string, []string) {
	// 帮助函数：如果提供了 logWriter，则将日志输出到该处
	log := func(format string, a ...interface{}) {
		msg := fmt.Sprintf(format, a...)
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		if logWriter != nil {
			logWriter.Write([]byte(msg))
		}
	}

	var repoTask models.Task
	res := database.DB.Where("id = ?", taskID).Limit(1).Find(&repoTask)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, nil
	}

	if repoTask.Type != constant.TaskTypeRepo {
		return nil, nil
	}

	var repoCfg models.RepoConfig
	if err := json.Unmarshal([]byte(repoTask.Config), &repoCfg); err != nil {
		return nil, nil
	}

	// 如果命令行强制开启，则覆盖配置
	if forceCommentToTask {
		repoCfg.CommentToTask = "true"
	}

	// 1. 确定解析策略
	strategy := GetParserStrategy(repoCfg.RepoSource)

	// 目标路径
	targetPath := repoCfg.TargetPath
	if targetPath == "" {
		targetPath = repoTask.WorkDir
	} else if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(utils.ResolveAbsScriptsDir(), targetPath)
	}
	if targetPath == "" {
		return nil, nil
	}
	targetPath = filepath.Clean(targetPath)

	// 获取仓库标识符
	repoId := utils.GetRepoIdentifier(repoCfg.SourceURL, repoCfg.Branch)

	gitDir := filepath.Join(targetPath, ".git")
	if !isDir(targetPath) || !pathExists(gitDir) {
		repoPath := filepath.Join(targetPath, repoId)
		if pathExists(repoPath) {
			targetPath = repoPath
		}
	}

	if !pathExists(targetPath) {
		return nil, nil
	}

	// 同步过程中使用的标签
	tag := fmt.Sprintf("%s", repoId)

	exts := getValidExtensions(repoCfg.Extensions)

	log("\n----------------------------------------")
	log("  开始扫描脚本并自动注册定时任务  ")
	log("----------------------------------------")

	foundSourceIDs := make(map[string]bool)
	var upsertedIDs []string
	var deletedIDs []string
	newTaskCount := 0
	updateTaskCount := 0

	filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if strings.Contains(path, ".git") {
			return nil
		}

		ext := filepath.Ext(path)
		if !strategy.SupportExtension(ext, exts) {
			return nil
		}

		// 2. 元数据提取
		taskName, taskCron := strategy.ExtractMeta(path, ext, repoCfg)

		// 3. 过滤处理
		relRepoPath, _ := filepath.Rel(targetPath, path)
		filename := filepath.Base(path)
		if !strategy.ShouldProcess(relRepoPath, filename, repoCfg) {
			return nil
		}

		if taskName != "" && taskCron != "" && repoCfg.AutoAddCron {
			// 获取脚本相对于数据目录的路径
			absScriptsDir := utils.ResolveAbsScriptsDir()
			// absTargetPath, _ := filepath.Abs(targetPath)
			absPath, _ := filepath.Abs(path)

			// 计算 SourceID: 相对于脚本目录的完整路径，并清洗特殊符号
			relPath, _ := filepath.Rel(absScriptsDir, absPath)
			sourceID := sanitizeIdentifier(relPath)

			// 替换绝对路径为代号 $SCRIPTS_DIR$
			displayPath := path
			displayWorkDir := targetPath
			if strings.HasPrefix(absPath, absScriptsDir) {
				// 命令仅使用文件名，工作目录设置为脚本所在目录
				displayPath = filepath.Base(path)
				relDir, _ := filepath.Rel(absScriptsDir, filepath.Dir(absPath))
				if relDir == "." {
					displayWorkDir = constant.ScriptsDirPlaceholder
				} else {
					displayWorkDir = filepath.ToSlash(filepath.Join(constant.ScriptsDirPlaceholder, relDir))
				}
			}

			// 找到任务，进行保存
			command := getCommandByExt(ext, displayPath)
			taskID, isNew := upsertRepoTask(&repoTask, sourceID, taskName, command, taskCron, displayWorkDir, tag)

			if isNew {
				log("[新增] 任务: %s (%s)", taskName, filename)
				newTaskCount++
			} else {
				log("[更新] 任务: %s (%s)", taskName, filename)
				updateTaskCount++
			}
			foundSourceIDs[sourceID] = true
			upsertedIDs = append(upsertedIDs, taskID)
		}

		return nil
	})

	// 清理该仓库下不再存在的旧脚本任务
	deletedTaskCount := 0
	var oldTasks []models.Task
	if err := database.DB.Where("repo_task_id = ?", repoTask.ID).Find(&oldTasks).Error; err == nil {
		for _, ot := range oldTasks {
			if !foundSourceIDs[ot.SourceID] {
				log("[移除] 脚本已不存在，删除对应任务: %s", ot.Name)
				deletedTaskCount++
				deletedIDs = append(deletedIDs, ot.ID)
				database.DB.Unscoped().Where("id = ?", ot.ID).Delete(&models.Task{})
			}
		}
	}

	log("\n扫描完成: [新增 %d] [更新 %d] [移除 %d]", newTaskCount, updateTaskCount, deletedTaskCount)
	log("----------------------------------------")
	return upsertedIDs, deletedIDs
}

// upsertRepoTask 处理来自仓库的任务的创建或更新
func upsertRepoTask(parentTask *models.Task, sourceID, name, command, cron, workDir, tag string) (string, bool) {
	defaultTaskConfig := `{"$task_all_envs":true,"$task_concurrency":0}`
	var existing models.Task
	tx := database.DB.Where("source_id = ? AND repo_task_id = ?", sourceID, parentTask.ID).Limit(1).Find(&existing)

	if tx.RowsAffected > 0 {
		// 更新操作
		existing.Name = name
		existing.Command = models.BigText(command)
		existing.Schedule = normalizeCron(cron)
		existing.Languages = parentTask.Languages
		existing.SourceID = sourceID
		existing.RepoTaskID = parentTask.ID
		existing.WorkDir = workDir
		// 如果原配置为空或者是 {}，则应用默认配置
		if string(existing.Config) == "" || string(existing.Config) == "{}" {
			existing.Config = models.BigText(defaultTaskConfig)
		}
		// 默认开启按条数清理30条
		if existing.CleanConfig == "" {
			existing.CleanConfig = `{"type":"count","keep":30}`
		}
		// 显式白名单模式：只更新脚本核心相关的字段，其他所有字段（如 Enabled, Pin, Remark 等）均不触碰
		database.DB.Model(&existing).
			Select("Name", "Command", "Schedule", "WorkDir", "Languages").
			Updates(&existing)
		return existing.ID, false
	} else {
		// 创建新任务
		newTask := &models.Task{
			Name:        name,
			Command:     models.BigText(command),
			Schedule:    normalizeCron(cron),
			Type:        "task",
			TriggerType: constant.TriggerTypeCron,
			Tags:        tag,
			Languages:   parentTask.Languages,
			Timeout:     parentTask.Timeout,
			Config:      models.BigText(defaultTaskConfig),
			Enabled:     utils.BoolPtr(true),
			WorkDir:     workDir,
			SourceID:    sourceID,
			RepoTaskID:  parentTask.ID,
			CleanConfig: `{"type":"count","keep":30}`,
		}
		newTask.ID = utils.GenerateID()
		database.DB.Create(newTask)

		// 确保将标签同步写入到新的 DataRelation 中
		if tag != "" {
			relation.DataRelation.SaveTags(newTask.ID, constant.RelationTypeTaskTag, tag)
		}
		
		return newTask.ID, true
	}
}
