package utils

import (
	"os"
	"runtime"
	"strings"
	"sync"
)

var (
	dockerCheckOnce      sync.Once
	isRunningInDockerVal bool
)

// IsRunningInDocker 检查当前进程是否在容器（Docker / Containerd / K8s / Podman）环境中运行。
// 检测方式依次包括：
// 1. 显式环境变量 (BH_IS_DOCKER=true / CONTAINER=docker)
// 2. 根目录容器标识文件 (/.dockerenv / /run/.containerenv)
// 3. Linux 下 cgroup 控制组特征签名 (/proc/1/cgroup, /proc/self/cgroup)
func IsRunningInDocker() bool {
	dockerCheckOnce.Do(func() {
		isRunningInDockerVal = checkRunningInDocker()
	})
	return isRunningInDockerVal
}

// IsInDocker 是 IsRunningInDocker 的兼容别名
func IsInDocker() bool {
	return IsRunningInDocker()
}

func checkRunningInDocker() bool {
	// 1. 环境变量标识（支持容器启动脚本显式注入或测试指定）
	if os.Getenv("BH_IS_DOCKER") == "true" || os.Getenv("CONTAINER") == "docker" {
		return true
	}

	// 2. 标准容器标识文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}

	// 3. Linux cgroup 特征检测
	if runtime.GOOS == "linux" {
		for _, p := range []string{"/proc/1/cgroup", "/proc/self/cgroup"} {
			if data, err := os.ReadFile(p); err == nil {
				content := string(data)
				if strings.Contains(content, "docker") || strings.Contains(content, "containerd") || strings.Contains(content, "kubepods") {
					return true
				}
			}
		}
	}

	return false
}
