//go:build windows

package processcontrol

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	jobObjectExtendedLimitInformation = 9
	jobObjectLimitKillOnJobClose      = 0x00002000
	processTerminate                  = 0x0001
	processSetQuota                   = 0x0100
	createSuspended                   = 0x00000004
	threadSuspendResume               = 0x0002
	th32csSnapThread                  = 0x00000004
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procThread32First            = kernel32.NewProc("Thread32First")
	procThread32Next             = kernel32.NewProc("Thread32Next")
	procOpenThread               = kernel32.NewProc("OpenThread")
	procResumeThread             = kernel32.NewProc("ResumeThread")
	closeWindowsHandle           = syscall.CloseHandle

	openProcessForJob = func(processID uint32) (syscall.Handle, error) {
		process, _, callErr := procOpenProcess.Call(processTerminate|processSetQuota, 0, uintptr(processID))
		if process == 0 {
			return 0, callErr
		}
		return syscall.Handle(process), nil
	}
	assignProcessToJob = func(job, process syscall.Handle) error {
		ok, _, callErr := procAssignProcessToJobObject.Call(uintptr(job), uintptr(process))
		if ok == 0 {
			return callErr
		}
		return nil
	}
)

type basicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type extendedLimitInformation struct {
	BasicLimitInformation basicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

type treeControl struct{ job syscall.Handle }

type threadEntry32 struct {
	Size           uint32
	Usage          uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePriority   int32
	DeltaPriority  int32
	Flags          uint32
}

func prepare(cmd *exec.Cmd) (*treeControl, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createSuspended}
	handle, _, callErr := procCreateJobObjectW.Call(0, 0)
	if handle == 0 {
		return nil, fmt.Errorf("create subprocess job object: %w", callErr)
	}
	control := &treeControl{job: syscall.Handle(handle)}
	info := extendedLimitInformation{}
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	ok, _, setErr := procSetInformationJobObject.Call(
		handle,
		jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
	)
	if ok == 0 {
		return nil, errors.Join(
			fmt.Errorf("configure subprocess job object: %w", setErr),
			control.close(),
		)
	}
	return control, nil
}

func (c *treeControl) attach(cmd *exec.Cmd) (result error) {
	process, openErr := openProcessForJob(uint32(cmd.Process.Pid))
	if openErr != nil {
		return fmt.Errorf("open subprocess for job assignment: %w", openErr)
	}
	defer func() { result = errors.Join(result, closeWindowsHandle(process)) }()
	if assignErr := assignProcessToJob(c.job, process); assignErr != nil {
		return fmt.Errorf("assign subprocess to job object: %w", assignErr)
	}
	return resumePrimaryThread(uint32(cmd.Process.Pid))
}

func resumePrimaryThread(processID uint32) (result error) {
	snapshot, _, snapshotErr := procCreateToolhelp32Snapshot.Call(th32csSnapThread, 0)
	if snapshot == ^uintptr(0) {
		return fmt.Errorf("snapshot subprocess threads: %w", snapshotErr)
	}
	defer func() { result = errors.Join(result, closeWindowsHandle(syscall.Handle(snapshot))) }()
	entry := threadEntry32{Size: uint32(unsafe.Sizeof(threadEntry32{}))}
	ok, _, firstErr := procThread32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ok == 0 {
		return fmt.Errorf("enumerate subprocess threads: %w", firstErr)
	}
	for {
		if entry.OwnerProcessID == processID {
			thread, _, openErr := procOpenThread.Call(threadSuspendResume, 0, uintptr(entry.ThreadID))
			if thread == 0 {
				return fmt.Errorf("open suspended subprocess thread: %w", openErr)
			}
			resumed, _, resumeErr := procResumeThread.Call(thread)
			closeErr := closeWindowsHandle(syscall.Handle(thread))
			if resumed == ^uintptr(0) {
				return errors.Join(fmt.Errorf("resume subprocess thread: %w", resumeErr), closeErr)
			}
			return closeErr
		}
		ok, _, nextErr := procThread32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ok == 0 {
			return fmt.Errorf("primary subprocess thread not found: %w", nextErr)
		}
	}
}

func (c *treeControl) terminate(cmd *exec.Cmd) error {
	if c.job == 0 {
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	ok, _, callErr := procTerminateJobObject.Call(uintptr(c.job), 1)
	if ok == 0 {
		return fmt.Errorf("terminate subprocess job object: %w", callErr)
	}
	return nil
}

func (c *treeControl) close() error {
	if c.job == 0 {
		return nil
	}
	err := closeWindowsHandle(c.job)
	c.job = 0
	if err != nil {
		return fmt.Errorf("close subprocess job object: %w", err)
	}
	return nil
}
