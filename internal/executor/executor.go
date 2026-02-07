package executor

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/utils"
)

// Task 任务基础接口
type Task interface {
	GetID() string
	GetName() string
	GetCommand() string
	GetTimeout() int
}

// CronTask 计划任务接口
type CronTask interface {
	Task
	GetSchedule() string
}

// Request 任务执行请求
type Request struct {
	Command string
	WorkDir string
	Envs    []string
	Timeout int // 分钟
}

// Result 任务执行结果
type Result struct {
	Output    string
	Status    string // success, failed
	Duration  int64  // 毫秒
	ExitCode  int
	StartTime time.Time
	EndTime   time.Time
}

// Hooks 执行钩子接口
type Hooks interface {
	// PreExecute 执行前钩子，返回日志ID和错误
	PreExecute(ctx context.Context, req Request) (logID uint, err error)

	// PostExecute 执行后钩子，处理日志压缩和记录更新
	PostExecute(ctx context.Context, logID uint, result *Result) error

	// OnHeartbeat 执行中心跳钩子，用于更新实时状态
	OnHeartbeat(ctx context.Context, logID uint, duration int64) error
}

// Execute 执行命令（基础版本，不带钩子）
func Execute(ctx context.Context, req Request, stdout, stderr io.Writer) (*Result, error) {
	return ExecuteWithHooks(ctx, req, stdout, stderr, nil)
}

// ExecuteWithHooks 执行命令（带钩子支持）
func ExecuteWithHooks(ctx context.Context, req Request, stdout, stderr io.Writer, hooks Hooks) (*Result, error) {
	start := time.Now()

	// 1. 执行前钩子
	var logID uint
	if hooks != nil {
		id, err := hooks.PreExecute(ctx, req)
		if err != nil {
			return &Result{
				Status:    "failed",
				Duration:  0,
				ExitCode:  1,
				StartTime: start,
				EndTime:   time.Now(),
			}, err
		}
		logID = id
	}

	// 2. 执行命令
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Minute)
	defer cancel()

	finalCommand := req.Command
	shell, args := utils.GetShellCommand(finalCommand)
	cmd := exec.CommandContext(execCtx, shell, args...)

	// 设置工作目录
	// 设置工作目录
	workDir := strings.TrimSpace(req.WorkDir)
	if workDir != "" {
		cmd.Dir = workDir
	}

	// 设置环境变量（始终继承系统环境变量）
	cmd.Env = os.Environ()
	if len(req.Envs) > 0 {
		cmd.Env = append(cmd.Env, req.Envs...)
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// 使用 cmd.Start() + Wait() 以便在后台处理心跳
	err := cmd.Start()
	if err != nil {
		// Start 失败的处理
		end := time.Now()
		result := &Result{
			Status:    "failed",
			Duration:  end.Sub(start).Milliseconds(),
			ExitCode:  1,
			StartTime: start, // 修正为 start
			EndTime:   end,
		}
		// 执行后钩子
		if hooks != nil {
			result.Output += "\n[System Error] " + err.Error()
			hooks.PostExecute(ctx, logID, result)
		}
		return result, err
	}

	// 启动心跳协程
	done := make(chan struct{})
	go func() {
		// 每3秒一次心跳
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if hooks != nil {
					hooks.OnHeartbeat(ctx, logID, time.Since(start).Milliseconds())
				}
			}
		}
	}()

	// 等待命令完成
	err = cmd.Wait()
	close(done) // 停止心跳

	end := time.Now()

	result := &Result{
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start).Milliseconds(),
	}

	if err != nil {
		result.Status = "failed"
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.Status = "success"
		result.ExitCode = 0
	}

	// 3. 执行后钩子
	if hooks != nil {
		if hookErr := hooks.PostExecute(ctx, logID, result); hookErr != nil {
			// 记录钩子错误但不影响执行结果
			result.Output += "\n[Hook Error] " + hookErr.Error()
		}
	}

	return result, err
}

// ParseEnvVars 解析环境变量字符串 "KEY1=VALUE1,KEY2=VALUE2"
func ParseEnvVars(envStr string) []string {
	if envStr == "" {
		return nil
	}

	pairs := strings.Split(envStr, ",")
	result := make([]string, 0, len(pairs))

	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		// 解码特殊字符
		pair = strings.ReplaceAll(pair, "{{COMMA}}", ",")
		pair = strings.ReplaceAll(pair, "{{EQUAL}}", "=")
		result = append(result, pair)
	}

	return result
}
