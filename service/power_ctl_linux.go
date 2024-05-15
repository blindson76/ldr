package service

/*
#include <unistd.h>
#include <sys/reboot.h>


*/
import "C"
import (
	"errors"
	"fmt"
	"log"
	"os/exec"
)

func (s *PowerCtl) LowInit() error {
	return nil
}
func (s *PowerCtl) Restart() error {

	log.Println("Restart")
	C.sync()
	res := C.reboot(C.RB_AUTOBOOT)
	return errors.New(fmt.Sprintf("Restart failed:%d", res))
}

func (s *PowerCtl) Shutdown() error {
	log.Println("Shutdown")

	C.sync()
	res := C.reboot(C.RB_POWER_OFF)
	return errors.New(fmt.Sprintf("Restart failed:%d", res))
}

func (s *PowerCtl) Logout() error {

	log.Println("Logout")
	// cmd := exec.Command("pkill", "-SIGKILL", "-u", "user")
	cmd := exec.Command("systemctl", "restart", "gdm.service")
	err := cmd.Run()
	if err != nil {
		return err
	} else {
		return nil
	}
}
