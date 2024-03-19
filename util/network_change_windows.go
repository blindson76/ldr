package util

import (
	"log"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modiphlpapi                               = windows.NewLazySystemDLL("iphlpapi.dll")
	procNotifyIpInterfaceChangeAddresses      = modiphlpapi.NewProc("NotifyIpInterfaceChange")
	procNotifyUnicastIpAddressChangeAddresses = modiphlpapi.NewProc("NotifyUnicastIpAddressChange")
	notifyHandle                              windows.Handle
)
var ()

func (n *NetworkChangeNotifier) init() {
	log.Println("init network change notifier")
	cb := syscall.NewCallback(func(ctx uintptr, row uintptr, changeType uintptr) uint64 {
		n.emits <- struct{}{}
		return 0
	})
	r1, _, err := procNotifyIpInterfaceChangeAddresses.Call(2, cb, uintptr(unsafe.Pointer(nil)), 1, uintptr(unsafe.Pointer(&notifyHandle)))
	if r1 != 0 {
		panic(err)
	}

	r1, _, err = procNotifyUnicastIpAddressChangeAddresses.Call(2, cb, uintptr(unsafe.Pointer(nil)), 1, uintptr(unsafe.Pointer(&notifyHandle)))
	if r1 != 0 {
		panic(err)
	}
	log.Println("Init done")
}
