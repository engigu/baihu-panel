//go:build windows

package windows

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procCreatePseudoConsole = modkernel32.NewProc("CreatePseudoConsole")
	procClosePseudoConsole  = modkernel32.NewProc("ClosePseudoConsole")
	procResizePseudoConsole = modkernel32.NewProc("ResizePseudoConsole")

	procInitializeProcThreadAttributeList = modkernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute          = modkernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList      = modkernel32.NewProc("DeleteProcThreadAttributeList")
)

const (
	PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE = 0x00020016
	EXTENDED_STARTUPINFO_PRESENT        = 0x00080000
)

type COORD struct {
	X int16
	Y int16
}

type STARTUPINFOEX struct {
	StartupInfo   windows.StartupInfo
	AttributeList uintptr
}

type ConPTYSession struct {
	hPC      uintptr
	hProcess windows.Handle
	hThread  windows.Handle
	inWrite  *os.File
	outRead  *os.File
	closed   bool
	mu       sync.Mutex
}

// HasConPTYSupport 自动检测当前 Windows 系统是否原生支持 ConPTY API (Win10 1809+ / Server 2019+)
func HasConPTYSupport() bool {
	return procCreatePseudoConsole.Find() == nil
}

// NewConPTYSession 创建一个新的 ConPTY 原生伪终端会话
func NewConPTYSession(cmdStr string, cols, rows uint16, env []string, dir string) (*ConPTYSession, error) {
	if !HasConPTYSupport() {
		return nil, fmt.Errorf("当前 Windows 系统不支持 ConPTY 原生伪终端")
	}

	// 1. 创建匿名输入/输出管道
	var inRead, inWrite windows.Handle
	var outRead, outWrite windows.Handle

	err := windows.CreatePipe(&inRead, &inWrite, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("创建输入管道失败: %w", err)
	}

	err = windows.CreatePipe(&outRead, &outWrite, nil, 0)
	if err != nil {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		return nil, fmt.Errorf("创建输出管道失败: %w", err)
	}

	// 2. 调用 CreatePseudoConsole 创建 ConPTY
	// 注意：Win64 ABI 下 COORD 是按值传递的 32 位结构 (uint32(cols) | (uint32(rows) << 16))
	coordVal := uintptr(uint32(cols) | (uint32(rows) << 16))
	var hPC uintptr

	res, _, _ := procCreatePseudoConsole.Call(
		coordVal,
		uintptr(inRead),
		uintptr(outWrite),
		0,
		uintptr(unsafe.Pointer(&hPC)),
	)
	if res != 0 {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		windows.CloseHandle(outWrite)
		return nil, fmt.Errorf("CreatePseudoConsole 失败: HRESULT 0x%X", res)
	}

	// 在 CreateProcess 成功启动子进程后再关闭 ConPTY 继承的端管道句柄
	defer windows.CloseHandle(inRead)
	defer windows.CloseHandle(outWrite)

	// 3. 配置 AttributeList (PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE)
	var sizeAttrList uintptr
	procInitializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&sizeAttrList)))

	attrList := make([]byte, sizeAttrList)
	r1, _, errList := procInitializeProcThreadAttributeList.Call(
		uintptr(unsafe.Pointer(&attrList[0])),
		1,
		0,
		uintptr(unsafe.Pointer(&sizeAttrList)),
	)
	if r1 == 0 {
		procClosePseudoConsole.Call(hPC)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, fmt.Errorf("InitializeProcThreadAttributeList 失败: %w", errList)
	}

	r1, _, errList = procUpdateProcThreadAttribute.Call(
		uintptr(unsafe.Pointer(&attrList[0])),
		0,
		PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		hPC,
		unsafe.Sizeof(hPC),
		0,
		0,
	)
	if r1 == 0 {
		procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrList[0])))
		procClosePseudoConsole.Call(hPC)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, fmt.Errorf("UpdateProcThreadAttribute 失败: %w", errList)
	}

	// 4. 配置 STARTUPINFOEX 并启动进程
	var siEx STARTUPINFOEX
	siEx.StartupInfo.Cb = uint32(unsafe.Sizeof(siEx))
	siEx.AttributeList = uintptr(unsafe.Pointer(&attrList[0]))

	var pi windows.ProcessInformation

	cmdPtr, err := windows.UTF16PtrFromString(cmdStr)
	if err != nil {
		procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrList[0])))
		procClosePseudoConsole.Call(hPC)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, err
	}

	var dirPtr *uint16
	if dir != "" {
		dirPtr, _ = windows.UTF16PtrFromString(dir)
	}

	// 环境变量转换
	var envBlock *uint16
	if len(env) > 0 {
		envBlock = createEnvBlock(env)
	}

	creationFlags := uint32(EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT)

	err = windows.CreateProcess(
		nil,
		cmdPtr,
		nil,
		nil,
		false,
		creationFlags,
		envBlock,
		dirPtr,
		&siEx.StartupInfo,
		&pi,
	)

	procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&attrList[0])))

	if err != nil {
		procClosePseudoConsole.Call(hPC)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, fmt.Errorf("CreateProcess ConPTY 进程启动失败: %w", err)
	}

	return &ConPTYSession{
		hPC:      hPC,
		hProcess: pi.Process,
		hThread:  pi.Thread,
		inWrite:  os.NewFile(uintptr(inWrite), "inWrite"),
		outRead:  os.NewFile(uintptr(outRead), "outRead"),
	}, nil
}

func (s *ConPTYSession) Read(p []byte) (n int, err error) {
	return s.outRead.Read(p)
}

func (s *ConPTYSession) Write(p []byte) (n int, err error) {
	return s.inWrite.Write(p)
}

func (s *ConPTYSession) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.hPC == 0 {
		return nil
	}
	coordVal := uintptr(uint32(cols) | (uint32(rows) << 16))
	res, _, _ := procResizePseudoConsole.Call(s.hPC, coordVal)
	if res != 0 {
		return fmt.Errorf("ResizePseudoConsole 失败: HRESULT 0x%X", res)
	}
	return nil
}

func (s *ConPTYSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true

	if s.inWrite != nil {
		s.inWrite.Close()
	}
	if s.outRead != nil {
		s.outRead.Close()
	}

	if s.hPC != 0 {
		procClosePseudoConsole.Call(s.hPC)
		s.hPC = 0
	}

	if s.hProcess != 0 {
		windows.TerminateProcess(s.hProcess, 1)
		windows.CloseHandle(s.hProcess)
		s.hProcess = 0
	}
	if s.hThread != 0 {
		windows.CloseHandle(s.hThread)
		s.hThread = 0
	}

	return nil
}

func createEnvBlock(env []string) *uint16 {
	if len(env) == 0 {
		return nil
	}
	var block []uint16
	for _, e := range env {
		u, err := windows.UTF16FromString(e)
		if err == nil {
			block = append(block, u...)
		}
	}
	block = append(block, 0)
	return &block[0]
}
