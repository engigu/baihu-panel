package executor

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/utils"
)

// Task 任务基础接口
type Task interface {
	GetID() string
	GetName() string
	GetCommand() string
	GetTimeout() int
	GetWorkDir() string
	GetEnvs() string
	GetLanguages() []map[string]string
	GetUseMise() bool
}

// CronTask 计划任务接口
type CronTask interface {
	Task
	GetSchedule() string
	UseMise() bool
}

// Request 任务执行请求
type Request struct {
	Command   string
	WorkDir   string
	Envs      []string
	Timeout   int // 任务超时时间（分钟）
	Languages []map[string]string
	UseMise   bool
}

// Result 任务执行结果
type Result struct {
	Output    string
	Error     string
	Status    string // 状态: success, failed
	Duration  int64  // 毫秒
	ExitCode  int
	StartTime time.Time
	EndTime   time.Time
}

// Hooks 执行钩子接口
type Hooks interface {
	// PreExecute 执行前钩子，返回日志ID和错误
	PreExecute(ctx context.Context, req Request) (logID string, err error)

	// PostExecute 执行后钩子，处理日志压缩和记录更新
	PostExecute(ctx context.Context, logID string, result *Result) error

	// OnHeartbeat 执行中心跳钩子，用于更新实时状态
	OnHeartbeat(ctx context.Context, logID string, duration int64) error
}

// Execute 执行命令（基础版本，不带钩子）
func Execute(ctx context.Context, req Request, stdout, stderr io.Writer) (*Result, error) {
	return ExecuteWithHooks(ctx, req, stdout, stderr, nil)
}

// ExecuteWithHooks 执行命令（带钩子支持）
func ExecuteWithHooks(ctx context.Context, req Request, stdout, stderr io.Writer, hooks Hooks) (*Result, error) {
	start := time.Now()

	// 演示模式拦截
	if constant.DemoMode {
		logger.Warnf("[Executor] 演示模式下已拦截命令执行: %s", req.Command)
		if stdout != nil {
			stdout.Write([]byte("\r\n\033[1;33m[演示模式] 命令执行已跳过\033[0m\r\n"))
		}

		// 仍然触发 PreExecute 以便流程完整
		var logID string
		if hooks != nil {
			logID, _ = hooks.PreExecute(ctx, req)
		}

		result := &Result{
			Status:    constant.TaskStatusFailed,
			Output:    "[演示模式] 该任务在演示模式下被禁用执行",
			StartTime: start,
			EndTime:   time.Now(),
		}

		if hooks != nil {
			hooks.PostExecute(ctx, logID, result)
		}
		return result, nil
	}

	// 2. 执行命令
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Minute)
	defer cancel()

	// 如果指定使用 mise，则预先构建好带 mise 的命令，这样 PreExecute 记录的就是完整命令
	if req.UseMise {
		req.Command = utils.BuildMiseCommand(req.Command, req.Languages)
		req.UseMise = false
	}

	// 1. 执行前钩子
	var logID string
	if hooks != nil {
		id, err := hooks.PreExecute(ctx, req)
		if err != nil {
			return &Result{
				Status:    constant.TaskStatusFailed,
				Duration:  0,
				ExitCode:  1,
				StartTime: start,
				EndTime:   time.Now(),
			}, err
		}
		logID = id
	}

	shell, args := utils.GetShellCommand(req.Command)
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
	// 强制注入终端环境标识及禁用输出缓冲的标志
	cmd.Env = append(cmd.Env,
		"TERM=xterm",
		"PYTHONUNBUFFERED=1",
		"NODE_NO_WARNINGS=1",
	)

	var pipeWriter *os.File
	var ptyFile *os.File
	var copyDone chan struct{}
	var err error

	var started bool
	// 尝试开启 PTY 模式（Unix/macOS 且输出合并时）
	if runtime.GOOS != "windows" && stdout != nil && (stdout == stderr || stdout == io.Discard) {
		// 强制注入终端环境标识及禁用输出缓冲的标志，确保 PTY 模式下最佳实时性能
		cmd.Env = append(cmd.Env,
			"TERM=xterm",
			"PYTHONUNBUFFERED=1",
			"NODE_NO_WARNINGS=1",
		)
		f, ptyErr := pty.Start(cmd)
		if ptyErr == nil {
			logger.Infof("[Executor] 任务 %s 启动于 PTY 模式", logID)
			ptyFile = f
			started = true
			copyDone = make(chan struct{})
			go func() {
				defer close(copyDone)
				// io.Copy 对于 PTY 来说是最稳健且即时的流式拷贝
				io.Copy(stdout, f)
				f.Close()
			}()
		} else {
			logger.Errorf("[Executor] 任务 %s PTY 启动失败: %v", logID, ptyErr)
		}
	}

	if !started {
		// 如果 stdout 和 stderr 指针不一致，但在逻辑上我们知道它们是同一个 MultiWriter，
		// 这里会显示为 Pipe 模式。
		if stdout != stderr && stdout != io.Discard {
			logger.Debugf("[Executor] 任务 %s stdout (%p) 和 stderr (%p) 不同，回退到 Pipe 模式。", logID, stdout, stderr)
		}
		logger.Infof("[Executor] 任务 %s 启动于 Pipe 模式", logID)
		if stdout != nil && stdout == stderr {
			pr, pw, err := os.Pipe()
			if err == nil {
				cmd.Stdout = pw
				cmd.Stderr = pw
				pipeWriter = pw
				copyDone = make(chan struct{})
				go func() {
					io.Copy(stdout, pr)
					pr.Close()
					close(copyDone)
				}()
			} else {
				cmd.Stdout = stdout
				cmd.Stderr = stderr
			}
		} else {
			cmd.Stdout = stdout
			cmd.Stderr = stderr
		}

		// 使用 cmd.Start() + Wait() 以便在后台处理心跳
		err = cmd.Start()
		if err != nil {
			if pipeWriter != nil {
				pipeWriter.Close()
			}
			// 启动失败的处理
			end := time.Now()
			result := &Result{
				Status:    constant.TaskStatusFailed,
				Duration:  end.Sub(start).Milliseconds(),
				ExitCode:  1,
				StartTime: start, // 记录开始时间
				EndTime:   end,
			}
			// 执行后钩子
			if hooks != nil {
				result.Output += "\n[系统错误] " + err.Error()
				hooks.PostExecute(ctx, logID, result)
			}
			return result, err
		}

		// 在父进程中关闭写端，这样子进程退出后 pr 才会收到 EOF
		if pipeWriter != nil {
			pipeWriter.Close()
		}
	} else {
		// PTY 模式下 cmd.Start() 已经在 pty.Start(cmd) 中调用过了
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

	// PTY 模式下需要显式关闭
	if ptyFile != nil {
		ptyFile.Close()
	}

	// 等待日志复制完成
	if copyDone != nil {
		<-copyDone
	}

	end := time.Now()

	result := &Result{
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start).Milliseconds(),
	}

	if err != nil {
		result.Status = constant.TaskStatusFailed
		result.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.Status = constant.TaskStatusSuccess
		result.ExitCode = 0
	}

	// 3. 执行后钩子
	if hooks != nil {
		if hookErr := hooks.PostExecute(ctx, logID, result); hookErr != nil {
			// 记录钩子错误但不影响执行结果
			result.Output += "\n[钩子错误] " + hookErr.Error()
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
