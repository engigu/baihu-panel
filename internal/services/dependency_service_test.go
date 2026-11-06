package services

import (
	"testing"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDependencyTestDB(t *testing.T) *DependencyService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Dependency{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	database.DB = db
	return NewDependencyService()
}

func TestDependencyService_CreateAndUpsert(t *testing.T) {
	ds := setupDependencyTestDB(t)

	// 1. 首次无版本号安装入库
	dep1 := &models.Dependency{
		Name:        "requests",
		Version:     "",
		Language:    "python",
		LangVersion: "3.11",
		Remark:      "初始安装",
	}
	if err := ds.Create(dep1); err != nil {
		t.Fatalf("首次入库失败: %v", err)
	}
	if dep1.ID == "" {
		t.Fatalf("预期生成 ID，实际为空")
	}
	firstID := dep1.ID

	// 2. 再次安装且自动探测出版本号，同环境应更新而非重复插入
	dep2 := &models.Dependency{
		Name:        "requests",
		Version:     "2.31.0",
		Language:    "python",
		LangVersion: "3.11",
		Remark:      "更新版本",
	}
	if err := ds.Create(dep2); err != nil {
		t.Fatalf("二次更新失败: %v", err)
	}
	if dep2.ID != firstID {
		t.Fatalf("预期复用已有记录 ID %s，实际为 %s", firstID, dep2.ID)
	}

	// 验证表中仅有一条记录且版本已更新
	list, err := ds.List("python", "3.11")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("预期只有 1 条记录，实际有 %d 条", len(list))
	}
	if list[0].Version != "2.31.0" {
		t.Fatalf("预期版本为 2.31.0，实际为 %s", list[0].Version)
	}
	if list[0].Remark != "更新版本" {
		t.Fatalf("预期备注为'更新版本'，实际为 %s", list[0].Remark)
	}

	// 3. 按已有 ID 重新安装/更新
	depReinstall := &models.Dependency{
		ID:          firstID,
		Name:        "requests",
		Version:     "2.32.0",
		Language:    "python",
		LangVersion: "3.11",
		Remark:      "重装更新",
	}
	if err := ds.Create(depReinstall); err != nil {
		t.Fatalf("按 ID 重装更新失败: %v", err)
	}

	listAfterReinstall, err := ds.List("python", "3.11")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(listAfterReinstall) != 1 {
		t.Fatalf("预期只有 1 条记录，实际有 %d 条", len(listAfterReinstall))
	}
	if listAfterReinstall[0].Version != "2.32.0" {
		t.Fatalf("预期版本为 2.32.0，实际为 %s", listAfterReinstall[0].Version)
	}

	// 4. 不同语言版本应作为新记录
	depOtherEnv := &models.Dependency{
		Name:        "requests",
		Version:     "2.28.0",
		Language:    "python",
		LangVersion: "3.10",
	}
	if err := ds.Create(depOtherEnv); err != nil {
		t.Fatalf("不同环境入库失败: %v", err)
	}

	allPython, err := ds.List("python", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(allPython) != 2 {
		t.Fatalf("预期共有 2 条 python 依赖，实际有 %d 条", len(allPython))
	}
}

func TestDependencyService_Delete(t *testing.T) {
	ds := setupDependencyTestDB(t)

	dep := &models.Dependency{
		Name:        "flask",
		Version:     "3.0.0",
		Language:    "python",
		LangVersion: "3.11",
	}
	if err := ds.Create(dep); err != nil {
		t.Fatalf("创建失败: %v", err)
	}

	if err := ds.Delete(dep.ID); err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	list, err := ds.List("python", "3.11")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("预期已删除，实际仍有 %d 条记录", len(list))
	}
}

