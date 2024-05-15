package util

import (
	"log"

	"golang.org/x/sys/unix"
)

func (n *NetworkChangeNotifier) init() {

	fd, err := unix.Socket(
		// Always used when opening netlink sockets.
		unix.AF_NETLINK,
		// Seemingly used interchangeably with SOCK_DGRAM,
		// but it appears not to matter which is used.
		unix.SOCK_RAW,
		// The netlink family that the socket will communicate
		// with, such as NETLINK_ROUTE or NETLINK_GENERIC.
		unix.NETLINK_ROUTE,
	)
	if err != nil {
		panic(err)
	}
	err = unix.Bind(fd, &unix.SockaddrNetlink{
		// Always used when binding netlink sockets.
		Family: unix.AF_NETLINK,
		// A bitmask of multicast groups to join on bind.
		// Typically set to zero.
		Groups: unix.RTMGRP_LINK | unix.RTMGRP_IPV4_IFADDR | unix.RTMGRP_IPV4_ROUTE,
		// If you'd like, you can assign a PID for this socket
		// here, but in my experience, it's easier to leave
		// this set to zero and let netlink assign and manage
		// PIDs on its own.
		Pid: 0,
	})
	if err != nil {
		panic(err)
	}
	go func() {
		n.emits <- struct{}{}
		data := make([]byte, 1024)
		for {
			_, _, err := unix.Recvfrom(fd, data, 0)
			if err == nil {
				n.emits <- struct{}{}
			}
		}
	}()
	log.Println("Init done")
}
