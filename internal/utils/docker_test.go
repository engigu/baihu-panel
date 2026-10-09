package utils

import (
	"os"
	"testing"
)

func TestCheckRunningInDocker(t *testing.T) {
	// 保存原环境变量
	origBH := os.Getenv("BH_IS_DOCKER")
	origContainer := os.Getenv("CONTAINER")
	defer func() {
		_ = os.Setenv("BH_IS_DOCKER", origBH)
		_ = os.Setenv("CONTAINER", origContainer)
	}()

	// 1. 测试 BH_IS_DOCKER=true
	_ = os.Setenv("BH_IS_DOCKER", "true")
	if !checkRunningInDocker() {
		t.Errorf("期望 checkRunningInDocker() 在 BH_IS_DOCKER=true 时返回 true")
	}

	// 2. 测试 CONTAINER=docker
	_ = os.Setenv("BH_IS_DOCKER", "")
	_ = os.Setenv("CONTAINER", "docker")
	if !checkRunningInDocker() {
		t.Errorf("期望 checkRunningInDocker() 在 CONTAINER=docker 时返回 true")
	}

	// 3. 验证 IsRunningInDocker 与 IsInDocker 接口一致性
	_ = os.Setenv("BH_IS_DOCKER", "")
	_ = os.Setenv("CONTAINER", "")
	res1 := IsRunningInDocker()
	res2 := IsInDocker()
	if res1 != res2 {
		t.Errorf("IsInDocker() (%v) 应与 IsRunningInDocker() (%v) 结果一致", res2, res1)
	}
}
