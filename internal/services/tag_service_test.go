package services

import (
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTagTestDB(t *testing.T) *TagService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存测试数据库失败: %v", err)
	}
	database.DB = db
	if err := db.AutoMigrate(&models.DataStorage{}, &models.DataRelation{}); err != nil {
		t.Fatalf("迁移标签表失败: %v", err)
	}
	return NewTagService()
}

// 验证 Issue #233：不同分类（任务标签 vs 环境变量）允许同名，但同一分类内部严格防重
func TestTagService_CreateTag_CrossTypeIsolation(t *testing.T) {
	ts := setupTagTestDB(t)

	// 1. 在任务标签分类下创建 "dev" 标签 -> 成功
	taskTag, err := ts.CreateTag("dev", constant.RelationTypeTaskTag)
	if err != nil {
		t.Fatalf("创建任务标签失败: %v", err)
	}
	if taskTag.Name != "dev" || taskTag.Type != constant.RelationTypeTaskTag {
		t.Fatalf("创建的任务标签属性不符: %+v", taskTag)
	}

	// 2. 在同一分类 (任务标签) 下再次创建 "dev" -> 应当报错 "标签名称已存在"
	_, err = ts.CreateTag("dev", constant.RelationTypeTaskTag)
	if err == nil || err.Error() != "标签名称已存在" {
		t.Fatalf("期望同分类重复创建报错'标签名称已存在'，实际返回: %v", err)
	}

	// 3. 在环境变量分类下创建相同名称 "dev" 标签 -> 应当成功（修复前会因全局查重误报错）
	envTag, err := ts.CreateTag("dev", constant.RelationTypeEnvTag)
	if err != nil {
		t.Fatalf("环境变量分类下创建同名标签失败 (Issue #233 复现): %v", err)
	}
	if envTag.Name != "dev" || envTag.Type != constant.RelationTypeEnvTag {
		t.Fatalf("创建的环境变量标签属性不符: %+v", envTag)
	}

	// 4. 在环境变量分类下再次创建 "dev" -> 应当报错 "标签名称已存在"
	_, err = ts.CreateTag("dev", constant.RelationTypeEnvTag)
	if err == nil || err.Error() != "标签名称已存在" {
		t.Fatalf("期望同分类重复创建报错'标签名称已存在'，实际返回: %v", err)
	}
}

// 验证 Issue #233：重命名标签时只在所属分类内查重，不同分类存在同名时不应被拦截
func TestTagService_RenameTag_CrossTypeIsolation(t *testing.T) {
	ts := setupTagTestDB(t)

	// 预置数据：
	// 任务标签分类已有 "prod"
	// 环境变量分类已有 "staging" 和 "test"
	_, err := ts.CreateTag("prod", constant.RelationTypeTaskTag)
	if err != nil {
		t.Fatalf("初始化任务标签失败: %v", err)
	}
	stagingEnv, err := ts.CreateTag("staging", constant.RelationTypeEnvTag)
	if err != nil {
		t.Fatalf("初始化环境变量标签 staging 失败: %v", err)
	}
	testEnv, err := ts.CreateTag("test", constant.RelationTypeEnvTag)
	if err != nil {
		t.Fatalf("初始化环境变量标签 test 失败: %v", err)
	}

	// 1. 将环境变量标签 staging 改名为 "prod"（该名称已在任务标签中存在）-> 应当成功！
	if err := ts.RenameTag(stagingEnv.ID, "prod"); err != nil {
		t.Fatalf("环境变量改名为任务已有同名标签失败 (Issue #233 重命名拦截复现): %v", err)
	}

	// 2. 将环境变量标签 test 也改名为 "prod"（同属环境变量分类且已有 prod）-> 应当报错
	if err := ts.RenameTag(testEnv.ID, "prod"); err == nil || err.Error() != "标签名称已存在" {
		t.Fatalf("同分类内改名为已存在名称应当拦截，实际返回: %v", err)
	}

	// 3. 标签重命名为自己当前的名字 -> 应当成功
	if err := ts.RenameTag(stagingEnv.ID, "prod"); err != nil {
		t.Fatalf("标签同名保持不变不应报错: %v", err)
	}
}

