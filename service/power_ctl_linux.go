package service

import (
	"log"
	"os/exec"
)

func (s *PowerCtl) Restart() error {

	log.Println("Restart")
	cmd := exec.Command("reboot")
	err := cmd.Run()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (s *PowerCtl) Shutdown() error {
	log.Println("Shutdown")
	cmd := exec.Command("shutdown", "-h", "now")
	err := cmd.Run()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (s *PowerCtl) Logout() error {

	log.Println("Logout")
	cmd := exec.Command("pkill", "-SIGKILL", "-u", "user")
	err := cmd.Run()
	if err != nil {
		return err
	} else {
		return nil
	}
}
func (s *PowerCtl) LowInit() error {
	return nil
}
