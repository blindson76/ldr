package service

import (
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type AnnounceService struct {
	ServiceInterface
	laddr *net.UDPAddr
	raddr *net.UDPAddr
	conn  *net.UDPConn
}

func (s *AnnounceService) Init(c *ServiceCtxt) error {

	raddr, err := net.ResolveUDPAddr("udp", "239.0.0.72:16644")
	if err != nil {
		panic(err)
	}
	s.raddr = raddr
	laddr, err := net.ResolveUDPAddr("udp", strings.Split(c.laddr.String(), "/")[0]+":0")
	if err != nil {
		panic(err)
	}
	s.laddr = laddr

	return nil
}
func (s *AnnounceService) Start(c *ServiceCtxt) error {

	ucon, err := net.DialUDP("udp", s.laddr, s.raddr)
	if err != nil {
		panic(err)
	}
	s.conn = ucon
	go s.work(c)
	return nil
}
func (s *AnnounceService) Stop(c *ServiceCtxt) error {
	return nil
}

func (s *AnnounceService) work(c *ServiceCtxt) error {
	for {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		res, err := s.conn.Write([]byte(hostname + "|" + c.listener.Addr().String()))
		if err != nil {
			log.Println("Exiting", res)
			break
		}
		time.Sleep(time.Second)
	}
	return nil
}
