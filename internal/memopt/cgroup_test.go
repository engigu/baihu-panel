package memopt

import "testing"

func TestParseCgroupFileCache(t *testing.T) {
	// 1. 测试 cgroup v2 格式
	v2Data := `anon 10485760
file 67108864
kernel_stack 123456
slab 2097152
`
	val, ok := parseCgroupFileCache(v2Data)
	if !ok || val != 67108864 {
		t.Errorf("cgroup v2 解析失败: 期望 67108864, 得到 %d, ok: %v", val, ok)
	}

	// 2. 测试 cgroup v1 格式 (包含 total_cache)
	v1Data := `cache 33554432
rss 10485760
total_cache 67108864
total_rss 10485760
`
	val, ok = parseCgroupFileCache(v1Data)
	if !ok || val != 67108864 {
		t.Errorf("cgroup v1 解析失败: 期望 67108864, 得到 %d, ok: %v", val, ok)
	}

	// 3. 测试 cgroup v2 包含 inactive_file 优先读取
	v2WithInactive := `anon 10485760
file 110940160
file_mapped 83886080
inactive_file 27054080
active_file 83886080
`
	val, ok = parseCgroupFileCache(v2WithInactive)
	if !ok || val != 27054080 {
		t.Errorf("cgroup v2 期望优先读取 inactive_file 27054080, 得到 %d, ok: %v", val, ok)
	}

	// 4. 测试 cgroup v2 仅含 file 与 file_mapped (扣除只读代码段)
	v2WithMapped := `anon 10485760
file 100000000
file_mapped 40000000
`
	val, ok = parseCgroupFileCache(v2WithMapped)
	if !ok || val != 60000000 {
		t.Errorf("cgroup v2 期望扣除 file_mapped 后得到 60000000, 得到 %d, ok: %v", val, ok)
	}

	// 5. 测试非法输入
	val, ok = parseCgroupFileCache("invalid content without key")
	if ok {
		t.Errorf("非法输入期望返回 ok=false, 得到 %v", ok)
	}
}

func TestCalculateDockerMemory(t *testing.T) {
	// 测试相减计算规则：totalUsage (95797248 B, ~91.36 MB) 扣除 inactive_file (74678272 B, ~71.22 MB)
	// 得到真实 Docker stats 内存占用 (21118976 B, ~20.14 MB)
	v2Stat := `anon 21118976
file 74678272
inactive_file 74678272
active_file 0
`
	dockerUsed, inactive, ok := CalculateDockerMemory(95797248, v2Stat)
	if !ok {
		t.Fatalf("CalculateDockerMemory 期望成功, 实际失败")
	}
	if inactive != 74678272 {
		t.Errorf("inactive_file 解析错误: 期望 74678272, 得到 %d", inactive)
	}
	if dockerUsed != 21118976 {
		t.Errorf("dockerUsed 相减规则计算错误: 期望 21118976, 得到 %d", dockerUsed)
	}

	// 边界测试：totalUsage 小于 inactive 时归 0
	dockerUsed, _, _ = CalculateDockerMemory(100, "inactive_file 200\n")
	if dockerUsed != 0 {
		t.Errorf("totalUsage 小于 inactive 时期望返回 0, 得到 %d", dockerUsed)
	}
}

func TestParseCgroupMemoryDetails(t *testing.T) {
	// 模拟真实服务器产生 115MB active_file 的 cgroup v2 账本
	v2Stat := `anon 12099584
file 123731968
inactive_file 8278016
active_file 115453952
slab_reclaimable 66716208
`
	totalUsage := uint64(202715008)
	stat := ParseCgroupMemoryDetails(totalUsage, v2Stat)

	if stat.InactiveFile != 8278016 {
		t.Errorf("期望 InactiveFile 为 8278016, 得到 %d", stat.InactiveFile)
	}
	if stat.ActiveFile != 115453952 {
		t.Errorf("期望 ActiveFile 为 115453952, 得到 %d", stat.ActiveFile)
	}
	if stat.TotalFileCache != 123731968 {
		t.Errorf("期望 TotalFileCache 为 123731968, 得到 %d", stat.TotalFileCache)
	}
	if stat.SlabReclaimable != 66716208 {
		t.Errorf("期望 SlabReclaimable 为 66716208, 得到 %d", stat.SlabReclaimable)
	}
	// DockerUsed 应当等于 202715008 - 8278016 = 194436992 (~185.4 MB)
	if stat.DockerUsedBytes != totalUsage-stat.InactiveFile {
		t.Errorf("DockerUsed 计算异常: 期望 %d, 得到 %d", totalUsage-stat.InactiveFile, stat.DockerUsedBytes)
	}
}
