package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
	"gopkg.in/ini.v1"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	shell32          = syscall.NewLazyDLL("shell32.dll")
	registerClassEx  = user32.NewProc("RegisterClassExW")
	createWindowEx   = user32.NewProc("CreateWindowExW")
	defWindowProc    = user32.NewProc("DefWindowProcW")
	destroyWindow    = user32.NewProc("DestroyWindow")
	postQuitMessage  = user32.NewProc("PostQuitMessage")
	createPopupMenu  = user32.NewProc("CreatePopupMenu")
	appendMenu       = user32.NewProc("AppendMenuW")
	trackPopupMenu   = user32.NewProc("TrackPopupMenu")
	getCursorPos     = user32.NewProc("GetCursorPos")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	shellNotifyIcon  = shell32.NewProc("Shell_NotifyIconW")
	createIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	getMessage       = user32.NewProc("GetMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessage  = user32.NewProc("DispatchMessageW")
	createMutex      = kernel32.NewProc("CreateMutexW")
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   syscall.Handle
	Icon       syscall.Handle
	Cursor     syscall.Handle
	Background syscall.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     syscall.Handle
}

type notifyIconData struct {
	Size             uint32
	Wnd              syscall.Handle
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             syscall.Handle
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GuidItem         syscall.GUID
	BalloonIcon      syscall.Handle
}

type point struct {
	X int32
	Y int32
}

const (
	wsOverlapped = 0x00000000
	wmCreate     = 0x0001
	wmDestroy    = 0x0002
	wmCommand    = 0x0111
	wmRbuttonup  = 0x0205
	wmUser       = 0x0400
	wmTrayIcon   = wmUser + 100

	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	mfString  = 0x00000000
	mfSeparator = 0x00000800
	mfChecked = 0x00000008
	mfUnchecked = 0x00000000

	tpmRightAlign = 0x0008
	tpmBottomAlign = 0x0020

	// 默认高位监听端口
	defaultPort = 38052
)

var (
	hwnd      syscall.Handle
	hmenu     syscall.Handle
	nid       notifyIconData
	panelCmd  *exec.Cmd
	panelUrl  = "http://localhost:38052"
	appName   = "BaihuPanelTray"
	regAutoStartKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
	activePid int // 专门用于精准记录和杀掉本托盘拉起的子进程
	mutexHandle syscall.Handle
)

func main() {
	runtime.LockOSThread()

	// 确保单实例运行 (单例模式)
	mutexName, _ := syscall.UTF16PtrFromString("Local\\BaihuPanelTrayMutex")
	ret, _, err := createMutex.Call(0, 1, uintptr(unsafe.Pointer(mutexName)))
	if err != nil && err.(syscall.Errno) == syscall.ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}
	mutexHandle = syscall.Handle(ret)
	defer syscall.CloseHandle(mutexHandle)

	// 启动面板服务
	startPanelService()

	className, _ := syscall.UTF16PtrFromString("BaihuPanelTrayClass")
	var wndClass wndClassEx
	wndClass.Size = uint32(unsafe.Sizeof(wndClass))
	wndClass.WndProc = syscall.NewCallback(wndProc)
	wndClass.ClassName = className

	registerClassEx.Call(uintptr(unsafe.Pointer(&wndClass)))

	hwndRet, _, _ := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(className)),
		wsOverlapped,
		0, 0, 0, 0,
		0, 0, 0, 0,
	)
	hwnd = syscall.Handle(hwndRet)

	// 创建托盘图标
	setupTrayIcon()

	// 消息循环
	var msg struct {
		Hwnd    syscall.Handle
		Message uint32
		Wparam  uintptr
		Lparam  uintptr
		Time    uint32
		Pt      point
	}

	for {
		ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}

	// 清理
	stopPanelService()
	shellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func setupTrayIcon() {
	nid.Size = uint32(unsafe.Sizeof(nid))
	nid.Wnd = hwnd
	nid.ID = 1001
	nid.Flags = nifMessage | nifIcon | nifTip
	nid.CallbackMessage = wmTrayIcon

	// 加载自定义图标
	icoData := getIconData()
	if len(icoData) > 22 {
		// 自动从 ICO 容器中解析真实的 PNG 数据偏移
		offset := uint32(icoData[18]) | (uint32(icoData[19]) << 8) | (uint32(icoData[20]) << 16) | (uint32(icoData[21]) << 24)
		size := uint32(icoData[14]) | (uint32(icoData[15]) << 8) | (uint32(icoData[16]) << 16) | (uint32(icoData[17]) << 24)
		
		if offset > 0 && offset + size <= uint32(len(icoData)) && string(icoData[offset:offset+8]) == "\x89PNG\r\n\x1a\n" {
			pngData := icoData[offset : offset+size]
			hIcon, _, _ := createIconFromResourceEx.Call(
				uintptr(unsafe.Pointer(&pngData[0])),
				uintptr(len(pngData)),
				1, // TRUE (Icon)
				0x00030000,
				0, // 自动使用 PNG 文件真实的尺寸比例 (例如 32x32)
				0,
				0,
			)
			nid.Icon = syscall.Handle(hIcon)
		} else {
			// 回退加载原生 ICO (例如无 PNG 容器的原始 BMP 格式)
			hIcon, _, _ := createIconFromResourceEx.Call(
				uintptr(unsafe.Pointer(&icoData[0])),
				uintptr(len(icoData)),
				1, // TRUE (Icon)
				0x00030000,
				16,
				16,
				0,
			)
			nid.Icon = syscall.Handle(hIcon)
		}
	}

	copy(nid.Tip[:], syscall.StringToUTF16("白虎面板 - 自动化任务调度平台"))
	shellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
}

func wndProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case wmTrayIcon:
		if lparam == wmRbuttonup {
			showPopupMenu()
		}
	case wmCommand:
		id := uint32(wparam & 0xFFFF)
		handleMenuCommand(id)
	case wmDestroy:
		postQuitMessage.Call(0)
	default:
		ret, _, _ := defWindowProc.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
		return ret
	}
	return 0
}

func showPopupMenu() {
	hmenuRet, _, _ := createPopupMenu.Call()
	hmenu = syscall.Handle(hmenuRet)

	openStr, _ := syscall.UTF16PtrFromString("🌐 打开面板")
	statusStrText := "🔴 状态：已停止"
	statusFlags := uint32(mfString | 0x00000001) // mfGrayed = 0x00000001
	if isServiceRunning() {
		statusStrText = "🟢 状态：运行中"
	}
	statusStr, _ := syscall.UTF16PtrFromString(statusStrText)
	restartStr, _ := syscall.UTF16PtrFromString("🔄 重启服务")
	configStr, _ := syscall.UTF16PtrFromString("⚙️ 打开配置文件")
	logStr, _ := syscall.UTF16PtrFromString("📄 查看运行日志")
	autoStr, _ := syscall.UTF16PtrFromString("📌 开机自启")
	exitStr, _ := syscall.UTF16PtrFromString("🚪 退出")

	appendMenu.Call(uintptr(hmenu), mfString, 1, uintptr(unsafe.Pointer(openStr)))
	appendMenu.Call(uintptr(hmenu), mfSeparator, 0, 0)
	appendMenu.Call(uintptr(hmenu), uintptr(statusFlags), 2, uintptr(unsafe.Pointer(statusStr)))
	appendMenu.Call(uintptr(hmenu), mfString, 3, uintptr(unsafe.Pointer(restartStr)))
	appendMenu.Call(uintptr(hmenu), mfString, 6, uintptr(unsafe.Pointer(configStr)))
	appendMenu.Call(uintptr(hmenu), mfString, 7, uintptr(unsafe.Pointer(logStr)))

	autoFlags := uint32(mfString)
	if isAutoStartEnabled() {
		autoFlags |= mfChecked
	} else {
		autoFlags |= mfUnchecked
	}
	appendMenu.Call(uintptr(hmenu), uintptr(autoFlags), 4, uintptr(unsafe.Pointer(autoStr)))

	appendMenu.Call(uintptr(hmenu), mfSeparator, 0, 0)
	appendMenu.Call(uintptr(hmenu), mfString, 5, uintptr(unsafe.Pointer(exitStr)))

	var pt point
	getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	setForegroundWindow.Call(uintptr(hwnd))
	trackPopupMenu.Call(
		uintptr(hmenu),
		tpmRightAlign|tpmBottomAlign,
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		uintptr(hwnd),
		0,
	)
}

func handleMenuCommand(id uint32) {
	switch id {
	case 1: // 打开面板
		openBrowser(panelUrl)
	case 3: // 重启服务
		stopPanelService()
		time.Sleep(500 * time.Millisecond)
		startPanelService()
	case 4: // 开机自启
		if isAutoStartEnabled() {
			if err := disableAutoStart(); err == nil {
				// Success
			}
		} else {
			if err := enableAutoStart(); err == nil {
				// Success
			}
		}
	case 5: // 退出
		destroyWindow.Call(uintptr(hwnd))
	case 6: // 打开配置文件
		openConfigFile()
	case 7: // 查看运行日志
		openLogFile()
	}
}

func startPanelService() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Dir(exePath)
	baihuExe := filepath.Join(dir, "baihu.exe")

	if _, err := os.Stat(baihuExe); os.IsNotExist(err) {
		baihuExe = "baihu.exe"
	}

	// 确认并解析配置文件路径，以支持以默认配置文件启动
	configPath := filepath.Clean(filepath.Join(dir, "configs", "config.ini"))
	configDir := filepath.Dir(configPath)

	// 如果没有 configs 目录，尝试回退到当前目录
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		_ = os.MkdirAll(configDir, 0755)
	}

	// 强制保证 config.ini 存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 读取项目中的模版 config.example.ini
		examplePath := filepath.Clean(filepath.Join(dir, "configs", "config.example.ini"))
		if _, err := os.Stat(examplePath); os.IsNotExist(err) {
			examplePath = "configs/config.example.ini"
		}

		// 使用 ini 包加载并设置默认端口，不改动源模板文件
		if cfg, err := ini.Load(examplePath); err == nil {
			cfg.Section("server").Key("port").SetValue(fmt.Sprintf("%d", defaultPort))
			_ = cfg.SaveTo(configPath)
		} else {
			// 兜底极简配置
			configContent := fmt.Sprintf("[server]\nport = %d\nhost = 0.0.0.0\n\n[database]\ntype = sqlite\npath = data/baihu.db\n\n[security]\nsecret = baihu_secret_key_change_me\n", defaultPort)
			_ = os.WriteFile(configPath, []byte(configContent), 0644)
		}
	}

	absConfigPath, err := filepath.Abs(configPath)
	if err == nil {
		configPath = absConfigPath
	}

	// 使用 ini 包安全且精准地读取 [server] 块内的端口
	port := defaultPort
	if cfg, err := ini.Load(configPath); err == nil {
		if pVal, err := cfg.Section("server").Key("port").Int(); err == nil && pVal > 0 {
			port = pVal
		}
	}

	panelUrl = fmt.Sprintf("http://localhost:%d", port)

	args := []string{"server", "--config", configPath}
	cmd := exec.Command(baihuExe, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	// 自动创建并重定向标准输出及标准错误到 data/logs/server.log，以便通过托盘直接查看
	logDir := filepath.Clean(filepath.Join(dir, "data", "logs"))
	_ = os.MkdirAll(logDir, 0755)
	logFilePath := filepath.Join(logDir, "server.log")
	if logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	_ = cmd.Start()
	panelCmd = cmd
	if cmd.Process != nil {
		// 精准记录拉起的子进程 PID，防止误杀其他程序
		activePid = cmd.Process.Pid
	}
}

func stopPanelService() {
	if panelCmd != nil && panelCmd.Process != nil {
		_ = panelCmd.Process.Kill()
		_ = panelCmd.Wait()
		panelCmd = nil
		activePid = 0
	} else if activePid > 0 {
		// 兜底：如果进程对象丢失但记录了 PID，进行精准终止
		if proc, err := os.FindProcess(activePid); err == nil {
			_ = proc.Kill()
			_, _ = proc.Wait()
		}
		activePid = 0
	}

	// 最终兜底：如果 PID 记录和 Cmd 控制柄均因异常丢失，通过 taskkill 确保没有残留程序死锁端口
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/F", "/IM", "baihu.exe").Run()
	}
}

func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func isAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regAutoStartKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(appName)
	return err == nil
}

func enableAutoStart() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, regAutoStartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(appName, fmt.Sprintf(`"%s"`, exePath))
}

func disableAutoStart() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regAutoStartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(appName)
}

func openConfigFile() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Dir(exePath)
	configPath := filepath.Clean(filepath.Join(dir, "configs", "config.ini"))

	// 确认文件存在，如果不存在则找一下当前目录或上级 configs
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = filepath.Clean(filepath.Join(dir, "config.ini"))
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// fallback 到工程目录的 configs/config.ini
			configPath = "configs/config.ini"
		}
	}

	absP, err := filepath.Abs(configPath)
	if err == nil {
		_ = exec.Command("notepad.exe", absP).Start()
	}
}

func openLogFile() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Dir(exePath)
	logDir := filepath.Clean(filepath.Join(dir, "data", "logs"))
	logFile := filepath.Join(logDir, "server.log")

	// 确认日志文件是否存在，存在直接用记事本打开最新的运行日志
	if _, err := os.Stat(logFile); err == nil {
		if absFile, err := filepath.Abs(logFile); err == nil {
			_ = exec.Command("notepad.exe", absFile).Start()
			return
		}
	}

	// 确保 logs 目录存在以防报错
	_ = os.MkdirAll(logDir, 0755)

	absP, err := filepath.Abs(logDir)
	if err == nil {
		// 回退：在 Windows 资源管理器中打开日志目录
		_ = exec.Command("explorer.exe", absP).Start()
	}
}

func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(h, &exitCode); err != nil {
		return false
	}
	return exitCode == 259 // 259 is STILL_ACTIVE
}

func isServiceRunning() bool {
	// 1. 如果 Cmd 内存实例依然正常存在，直接通过系统状态核验
	if panelCmd != nil && panelCmd.Process != nil {
		return isProcessRunning(panelCmd.Process.Pid)
	}

	// 2. 如果 Cmd 丢了但是 activePid 变量存有值，根据 PID 精准校验
	if activePid > 0 {
		return isProcessRunning(activePid)
	}
	return false
}

func getIconData() []byte {
	return defaultIcon
}
