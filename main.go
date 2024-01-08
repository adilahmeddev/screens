package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"time"
	"unsafe"

	w32 "github.com/jcollie/w32"
)

type Monitor struct {
	Name string
	Win int
	Mac int
	Height int
}
type Rectangle struct{
	Top, Left, Right, Bottom  int32
}
var (
	user32, _                              = syscall.LoadLibrary("User32.dll")
	dxva2, _                               = syscall.LoadLibrary("dxva2.dll")
	procSetVCPFeature, _                   = syscall.GetProcAddress(dxva2, "SetVCPFeature")
	procGetVCPFeatureAndVCPFeatureReply, _ = syscall.GetProcAddress(dxva2, "GetVCPFeatureAndVCPFeatureReply")
	procEnumDisplayMonitors, _             = syscall.GetProcAddress(user32, "EnumDisplayMonitors")
	monitors                               = map[string]Monitor{
		"asus": {
			Name: "asus",
			Win: 15,
			Mac: 17,
			Height: 1080,
		},
		"aoc": {
			Name: "aoc",
			Win: 15,
			Mac: 16,
			Height: 1440,
		},
	}
)

type mont struct {
	hnd    syscall.Handle
	height int
}

func main() {
	logBuffer := bytes.NewBuffer([]byte{})
	defer WriteLogToFile(logBuffer)

	args := os.Args[1:]
	logBuffer.WriteString(fmt.Sprintln("args: ", os.Args))

	if len(args) == 0 {
		for _, monitor := range monitors {
			if err := ToggleMonitor(monitor, logBuffer); err != nil {
				logBuffer.WriteString(fmt.Sprintf("Error: %q\n", err.Error()))
				return
			}
		}
		return
	}

	monitor, ok := monitors[args[0]]
	if !ok {
		logBuffer.WriteString(fmt.Sprintf("Error: arg %q is not a valid monitor\n", args[0]))
	}

	if err := ToggleMonitor(monitor, logBuffer); err != nil {
		logBuffer.WriteString(fmt.Sprintf("Error: %q\n", err.Error()))
		return
	}

}

func ToggleMonitor(monitor Monitor, output *bytes.Buffer) error {
	output.WriteString(fmt.Sprintf("Begin toggling %v\n", monitor.Name))
	start := time.Now()

	hmon := mont{}
	fn := syscall.NewCallback(func(hmonitor syscall.Handle, hdc syscall.Handle, rect *Rectangle, lparam uintptr) uintptr {
		if rect.Bottom == int32(monitor.Height) {
			hmon = mont{
				hnd:    hmonitor,
				height: int(rect.Bottom),
			}
		}
		return 1
	})

	_, _, callErr := syscall.SyscallN(procEnumDisplayMonitors, uintptr(0), uintptr(unsafe.Pointer(nil)), fn, uintptr(0))
	if callErr != 0 {
		return fmt.Errorf("%s", callErr.Error())
	}

	_, monitorCount := w32.GetNumberOfPhysicalMonitorsFromHMONITOR(w32.HMONITOR(hmon.hnd))

	physicalMonitorsBuffer := make([]w32.PHYSICAL_MONITOR, monitorCount)

	w32.GetPhysicalMonitorsFromHMONITOR(w32.HMONITOR(hmon.hnd), physicalMonitorsBuffer)

	currentVCPVal, err := GetVCPFeature(syscall.Handle(physicalMonitorsBuffer[0].Monitor), 0x60)
	if err != nil {
		return fmt.Errorf("Getvcp fail: %w", err)
	}

	newVCPVal := 0
	if currentVCPVal != uintptr(monitor.Win) {
		newVCPVal = monitor.Win
	} else {
		newVCPVal = monitor.Mac	
	}
	

	if err := SetVCPFeature(syscall.Handle(physicalMonitorsBuffer[0].Monitor), 0x60, newVCPVal); err != nil {
		return fmt.Errorf("set vcp failed: %w", err)
	}

	output.WriteString(fmt.Sprintf("Duration: %v\n", time.Now().Sub(start)))
	return nil

}
func SetVCPFeature(hPhysicalMonitor syscall.Handle, bVCPCode byte, value int) (err error) {
	_, _, callErr := syscall.SyscallN(procSetVCPFeature,
		uintptr(hPhysicalMonitor),
		uintptr(bVCPCode),
		uintptr(value),
	)
	if callErr != 0 {
		return fmt.Errorf(callErr.Error())
	}
	return nil
}
func GetVCPFeature(hPhysicalMonitor syscall.Handle, bVCPCode byte) (p uintptr, err error) {
	var out uint32

	_, _, callErr := syscall.SyscallN(procGetVCPFeatureAndVCPFeatureReply,
		uintptr(hPhysicalMonitor),
		uintptr(bVCPCode),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(&out)),
		uintptr(unsafe.Pointer(nil)),
	)
	if callErr != 0 {
		return 0, fmt.Errorf(callErr.Error())
	}
	return uintptr(out), nil
}

func WriteLogToFile(b *bytes.Buffer) {
	os.WriteFile(
		fmt.Sprintf(
			"log%v.txt",
			time.Now().Format("2006-01-02_15-14-05.9999")),
		b.Bytes(),
		fs.ModeAppend,
	)
}
