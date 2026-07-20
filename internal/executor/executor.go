package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/sandbox"
	"github.com/engigu/baihu-panel/internal/utils"
)

// Task 任务基础接口
type Task interface {
	GetID() string
	GetName() string
	GetCommand() string
	GetPreCommand() string
	GetPostCommand() string
	GetTimeout() int
	GetWorkDir() string
	GetEnvs() string
	GetEnvVars() []string
	GetLanguages() []map[string]string
	GetUseMise() bool
	GetSandboxProfileID() *string
}

// CronTask 计划任务接口
type CronTask interface {
	Task
	GetSchedule() string
	UseMise() bool
	GetSecrets() []string
	GetRandomRange() int
}

// Request 任务执行请求
type Request struct {
	Command        string
	PreCommand     string
	PostCommand    string
	WorkDir        string
	Envs           []string
	Timeout        int // 任务超时时间（分钟）
	Languages      []map[string]string
	UseMise        bool
	Sandbox        *models.SandboxConfig
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
	var execCtx context.Context
	var cancel context.CancelFunc

	if timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Minute)
	} else {
		execCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// 如果指定使用 mise，则预先构建好带 mise 的命令，这样 PreExecute 记录的就是完整命令
	if req.UseMise {
		utils.InjectNodePath(&req.Envs, req.Languages)
		req.Command = utils.BuildMiseCommand(req.Command, req.Languages)
		req.UseMise = false
	}

	// 组合指令（如果存在前置或后置指令）
	if req.PreCommand != "" || req.PostCommand != "" {
		finalCmd := ""
		if req.PreCommand != "" {
			finalCmd += req.PreCommand + "\n"
		}
		finalCmd += req.Command
		if req.PostCommand != "" {
			finalCmd += "\n" + req.PostCommand
		}
		req.Command = finalCmd
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
	
	// 校验并在 Unix 下使用 chpst 包装命令来实现 100% 稳定的降权与内存、并发限制
	if runtime.GOOS != "windows" && req.Sandbox != nil && req.Sandbox.UseSandbox {
		// 检查宿主机上是否存在 chpst 命令
		if _, chpstErr := exec.LookPath("chpst"); chpstErr == nil {
			var chpstArgs []string
			// 1. 降权用户
			chpstArgs = append(chpstArgs, "-u", fmt.Sprintf("%d:%d", req.Sandbox.UID, req.Sandbox.GID))
			// 2. 内存限制 (MB 转 Bytes)
			if req.Sandbox.MemoryLimit > 0 {
				chpstArgs = append(chpstArgs, "-m", fmt.Sprintf("%d", req.Sandbox.MemoryLimit*1024*1024))
			}
			// 3. 最大并发进程数限制
			if req.Sandbox.NprocLimit > 0 {
				chpstArgs = append(chpstArgs, "-p", fmt.Sprintf("%d", req.Sandbox.NprocLimit))
			}
			
			// 4. 重组原始执行参数，用 chpst 拉起 shell
			chpstArgs = append(chpstArgs, shell)
			chpstArgs = append(chpstArgs, args...)
			
			cmd := exec.CommandContext(execCtx, "chpst", chpstArgs...)
			usePty := stdout != nil && (stdout == stderr || stdout == io.Discard)
			SetProcessGroupAndCancel(cmd, usePty)
			
			// 在 chpst 接管降权后，我们就不需要 Go SysProcAttr 进行二次降权了，只应用命名空间隔离挂载
			sandbox.Apply(cmd, req.Sandbox)
			
			// 设置工作目录
			workDir := strings.TrimSpace(req.WorkDir)
			if workDir != "" {
				cmd.Dir = workDir
			}
			// 设置环境变量
			cmd.Env = os.Environ()
			if len(req.Envs) > 0 {
				cmd.Env = append(cmd.Env, req.Envs...)
			}
			cmd.Env = append(cmd.Env, "TERM=xterm", "PYTHONUNBUFFERED=1", "NODE_NO_WARNINGS=1")
			
			// 将最终实例赋值给原来的变量，继续走后续的 PTY / Pipe 流式日志捕获
			return executeCmdInstance(execCtx, cmd, logID, start, stdout, stderr, hooks, usePty)
		}
	}

	cmd := exec.CommandContext(execCtx, shell, args...)

	usePty := runtime.GOOS != "windows" && stdout != nil && (stdout == stderr || stdout == io.Discard)
	SetProcessGroupAndCancel(cmd, usePty)

	// 应用沙箱配置 (资源与特权控制)
	sandbox.Apply(cmd, req.Sandbox)

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

	return executeCmdInstance(execCtx, cmd, logID, start, stdout, stderr, hooks, usePty)
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
		pair = strings.ReplaceAll(pair, "{{NL}}", "\n")
		result = append(result, pair)
	}

	return result
}

// FormatEnvVars 将环境变量列表格式化为逗号分隔的字符串 "KEY1=VALUE1,KEY2=VALUE2"
// 会对 , 和 = 以及换行符进行转义
func FormatEnvVars(envs []string) string {
	if len(envs) == 0 {
		return ""
	}

	pairs := make([]string, 0, len(envs))
	for _, pair := range envs {
		// 寻找第一个等号
		idx := strings.Index(pair, "=")
		if idx == -1 {
			continue
		}
		name := pair[:idx]
		value := pair[idx+1:]

		// 转义特殊字符
		encodedValue := strings.ReplaceAll(value, ",", "{{COMMA}}")
		encodedValue = strings.ReplaceAll(encodedValue, "=", "{{EQUAL}}")
		encodedValue = strings.ReplaceAll(encodedValue, "\n", "{{NL}}")
		pairs = append(pairs, fmt.Sprintf("%s=%s", name, encodedValue))
	}

	return strings.Join(pairs, ",")
}

// executeCmdInstance 执行装配好的 Cmd 实例，供 chpst 包装或原生命令调用
func executeCmdInstance(ctx context.Context, cmd *exec.Cmd, logID string, start time.Time, stdout, stderr io.Writer, hooks Hooks, usePty bool) (*Result, error) {
	var pipeWriter *os.File
	var ptyFile *os.File
	var copyDone chan struct{}
	var err error

	var started bool
	// 尝试开启 PTY 模式（Unix/macOS 且输出合并时）
	if usePty && runtime.GOOS != "windows" && stdout != nil && (stdout == stderr || stdout == io.Discard) {
		f, ptyErr := pty.Start(cmd)
		if ptyErr == nil {
			logger.Infof("[Executor] #%s 启动于 PTY 模式", logID)
			ptyFile = f
			started = true
			copyDone = make(chan struct{})
			go func() {
				defer close(copyDone)
				io.Copy(stdout, f)
				f.Close()
			}()
		} else {
			logger.Errorf("[Executor] 任务 #%s PTY 启动失败: %v", logID, ptyErr)
		}
	}

	if !started {
		if stdout != stderr && stdout != io.Discard {
			logger.Debugf("[Executor] 任务 #%s stdout (%p) 和 stderr (%p) 不同，回退到 Pipe 模式。", logID, stdout, stderr)
		}
		logger.Infof("[Executor] #%s 启动于 Pipe 模式", logID)
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

		err = cmd.Start()
		if err != nil {
			if pipeWriter != nil {
				pipeWriter.Close()
			}
			end := time.Now()
			result := &Result{
				Status:    constant.TaskStatusFailed,
				Duration:  end.Sub(start).Milliseconds(),
				ExitCode:  1,
				StartTime: start,
				EndTime:   end,
			}
			if hooks != nil {
				result.Output += "\n[系统错误] " + err.Error()
				hooks.PostExecute(ctx, logID, result)
			}
			return result, err
		}

		if pipeWriter != nil {
			pipeWriter.Close()
		}
	}

	done := make(chan struct{})
	go func() {
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

	err = cmd.Wait()
	close(done)

	if ptyFile != nil {
		ptyFile.Close()
	}

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

	if hooks != nil {
		if hookErr := hooks.PostExecute(ctx, logID, result); hookErr != nil {
			result.Output += "\n[钩子错误] " + hookErr.Error()
		}
	}

	return result, err
}
