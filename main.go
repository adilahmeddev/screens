package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"time"
	"unsafe"
)

type Monitor struct {
	Name   string
	Win    int
	Mac    int
	Height int
}
type Rectangle struct {
	Top, Left, Right, Bottom int32
}

var (
	user32, _                                      = syscall.LoadLibrary("User32.dll")
	dxva2, _                                       = syscall.LoadLibrary("dxva2.dll")
	procSetVCPFeature, _                           = syscall.GetProcAddress(dxva2, "SetVCPFeature")
	procGetVCPFeatureAndVCPFeatureReply, _         = syscall.GetProcAddress(dxva2, "GetVCPFeatureAndVCPFeatureReply")
	procEnumDisplayMonitors, _                     = syscall.GetProcAddress(user32, "EnumDisplayMonitors")
	procGetPhysicalMonitorsFromHMONITOR, _         = syscall.GetProcAddress(dxva2, "GetPhysicalMonitorsFromHMONITOR")
	procGetNumberOfPhysicalMonitorsFromHMONITOR, _ = syscall.GetProcAddress(dxva2, "GetNumberOfPhysicalMonitorsFromHMONITOR")
	monitors                                       = map[string]Monitor{
		"asus": {
			Name:   "asus",
			Win:    15,
			Mac:    17,
			Height: 1080,
		},
		"aoc": {
			Name:   "aoc",
			Win:    15,
			Mac:    16,
			Height: 1440,
		},
	}
)

type PhysicalMonitor struct {
	Monitor     syscall.Handle
	Description [128]uint16
}
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
		output.WriteString(fmt.Sprintln(hmon))
		return 1
	})

	_, _, callErr := syscall.SyscallN(procEnumDisplayMonitors, uintptr(0), uintptr(unsafe.Pointer(nil)), fn, uintptr(0))
	if callErr != 0 {
		return fmt.Errorf("%s", callErr.Error())
	}

	monitorCount := 0
	_, _, callErr = syscall.SyscallN(procGetNumberOfPhysicalMonitorsFromHMONITOR, uintptr(hmon.hnd), uintptr(unsafe.Pointer(&monitorCount)))
	if callErr != 0 {
		return fmt.Errorf("%s", callErr.Error())
	}
	output.WriteString(fmt.Sprintf("monitor count: %v\n", monitorCount))

	physicalMonitorsBuffer := make([]PhysicalMonitor, monitorCount)
	syscall.SyscallN(procGetPhysicalMonitorsFromHMONITOR, uintptr(hmon.hnd), uintptr(len(physicalMonitorsBuffer)), uintptr(unsafe.Pointer(&physicalMonitorsBuffer[0])))

	output.WriteString(fmt.Sprint(physicalMonitorsBuffer))
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
	log := false
	for _, arg := range os.Args {
		if arg == "log" {
			log = true
		}
	}
	if !log {
		return
	}
	os.WriteFile(
		fmt.Sprintf(
			"log%v.txt",
			time.Now().Format("2006-01-02_15-14-05.9999")),
		b.Bytes(),
		fs.ModeAppend,
	)
}
