package util

import (
	"net"
	"strings"
)

func InterfaceByAddress(net_addr string) (*net.Interface, net.Addr) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}
	for i := 0; i < len(ifs); i++ {
		addrs, err := ifs[i].Addrs()
		if err != nil {
			continue
		}
		for ai := 0; ai < len(addrs); ai++ {
			addr := addrs[ai]
			if strings.HasPrefix(addr.String(), net_addr) {
				return &ifs[i], addr
			}
		}
	}
	return nil, nil
}
