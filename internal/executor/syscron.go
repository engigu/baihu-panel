package executor

import (
	"sync"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/systime"
	"github.com/robfig/cron/v3"
)

type SysCronManager struct {
	cron *cron.Cron
}

var (
	sysCronInstance *SysCronManager
	sysCronOnce     sync.Once
)


// InitSysCron 初始化系统的内部定时器
func InitSysCron(){
	GetSysCron()
}

// GetSysCron 获取内部系统定时器服务单例
func GetSysCron() *SysCronManager {
	sysCronOnce.Do(func() {
		// 使用秒级精度，指定为东八区
		c := cron.New(cron.WithSeconds(), cron.WithLocation(systime.CST))
		c.Start()
		sysCronInstance = &SysCronManager{
			cron: c,
		}
		logger.Infof("[SysCron] 内部系统定时管理器已启动")
	})
	return sysCronInstance
}

// AddJob 添加内部系统任务，spec为cron表达式（支持 @every 30s 这种快捷方式）
func (s *SysCronManager) AddJob(spec string, cmd func()) (cron.EntryID, error) {
	id, err := s.cron.AddFunc(spec, cmd)
	if err != nil {
		logger.Errorf("[SysCron] 无法添加系统任务: %s, err: %v", spec, err)
		return 0, err
	}
	return id, nil
}

// AddJobWithRun 立即开启一个协程异步执行一次任务，随后将其加入到系统定时任务中
func (s *SysCronManager) AddJobWithRun(spec string, cmd func()) (cron.EntryID, error) {
	// 立即异步执行一次
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("[SysCron] 立即执行任务时发生 panic: %v", r)
			}
		}()
		cmd()
	}()
	
	// 然后加入定时器
	return s.AddJob(spec, cmd)
}

// RemoveJob 动态移除指定的系统定时任务
func (s *SysCronManager) RemoveJob(id cron.EntryID) {
	s.cron.Remove(id)
}
