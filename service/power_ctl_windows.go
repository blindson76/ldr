package service

import (
	"C"
	"errors"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/sys/windows"
)
import (
	"syscall"
	"unsafe"
)

const (
	errnoERROR_IO_PENDING = 997
)

var (
	modadvapi32           = windows.NewLazySystemDLL("advapi32.dll")
	procInitiateShutdownW = modadvapi32.NewProc("InitiateShutdownW")
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
	errERROR_EINVAL     error = syscall.EINVAL
)

func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return errERROR_EINVAL
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}
func InitiateShutdownW(machineName *uint16, message *uint16, timeout uint32, shutdownFlags uint32, reason uint32) (err error) {
	r1, _, e1 := syscall.SyscallN(procInitiateShutdownW.Addr(), uintptr(unsafe.Pointer(machineName)), uintptr(unsafe.Pointer(message)), uintptr(timeout), uintptr(shutdownFlags), uintptr(reason))
	if r1 == 0 {
		err = errnoErr(e1)
	}
	log.Println("call:", r1, e1)
	return err
}

func (s *PowerCtl) Restart2() error {
	// err = InitiateShutdownW(nil, nil, 0, 0x1, 0)
	// return err
	return nil
}

func (s *PowerCtl) Restart() error {
	log.Println("Restart win")
	cmd := exec.Command("shutdown", "/r", "/f", "/t", "0")
	err := cmd.Run()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (s *PowerCtl) Shutdown() error {
	log.Println("Shutdown win")
	cmd := exec.Command("shutdown", "/s", "/f", "/t", "0")
	err := cmd.Run()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (s *PowerCtl) Logout() error {
	log.Println("Logoff win")
	out, err := exec.Command("query", "session").Output()
	if err != nil {
		lines := strings.Split(string(out), "\n")
		re := regexp.MustCompile(" +")
		for _, line := range lines {
			// log.Println("Line:", line)
			sess := re.Split(line, -1)
			// log.Println("Sess:", strings.Join(sess, "|"))
			if len(sess) > 4 && sess[4] == "Active" {
				cmd := exec.Command("logoff", sess[3])
				err := cmd.Run()
				if err != nil {
					return err
				} else {
					return nil
				}
			}

		}
	}
	return errors.New("logoff failed")
}

func (s *PowerCtl) LowInit() error {

	var token windows.Token
	tkp := windows.Tokenprivileges{}
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token)
	if err != nil {
		return err
	}
	//SeSystemEnvironmentPrivilege
	priv, err := windows.UTF16PtrFromString("SeShutdownPrivilege")
	if err != nil {
		return err
	}
	err = windows.LookupPrivilegeValue(nil, priv, &tkp.Privileges[0].Luid)
	if err != nil {
		return err
	}
	tkp.PrivilegeCount = 1
	tkp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	err = windows.AdjustTokenPrivileges(token, false, &tkp, 0, nil, nil)
	if err != nil {
		return err
	}
	priv, err = windows.UTF16PtrFromString("SeSystemEnvironmentPrivilege")
	if err != nil {
		return err
	}
	err = windows.LookupPrivilegeValue(nil, priv, &tkp.Privileges[0].Luid)
	if err != nil {
		return err
	}
	tkp.PrivilegeCount = 1
	tkp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	err = windows.AdjustTokenPrivileges(token, false, &tkp, 0, nil, nil)
	if err == nil {
		log.Println("GOT ENVVAR privilege")
	}
	return nil
}

func (s *PowerCtl) Stop() error {
	return nil
}
