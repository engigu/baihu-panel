package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
)

// GenerateMachineID 生成机器识别码
func GenerateMachineID() string {
	var parts []string

	// 主机名
	if hostname, err := os.Hostname(); err == nil {
		parts = append(parts, hostname)
	}

	// 获取所有非回环网卡的 MAC 地址，排序后取第一个（最稳定）
	if interfaces, err := net.Interfaces(); err == nil {
		var macs []string
		for _, iface := range interfaces {
			// 跳过回环接口、没有 MAC 地址的接口、虚拟接口
			if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
				continue
			}
			// 跳过 docker/veth 等虚拟网卡
			name := strings.ToLower(iface.Name)
			if strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "veth") ||
				strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "virbr") {
				continue
			}
			macs = append(macs, iface.HardwareAddr.String())
		}
		sort.Strings(macs)
		// 只使用第一个 MAC 地址（最稳定）
		if len(macs) > 0 {
			parts = append(parts, macs[0])
		}
	}

	// 操作系统和架构
	parts = append(parts, runtime.GOOS, runtime.GOARCH)

	data := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
