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

	// 3. 测试非法输入
	val, ok = parseCgroupFileCache("invalid content without key")
	if ok {
		t.Errorf("非法输入期望返回 ok=false, 得到 %v", ok)
	}
}
