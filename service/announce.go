package service

import (
	"log"
	"net"
	"os"
	"strings"
	"time"
	"tr/com/havelsan/hloader/util"
)

type AnnounceService struct {
	util.NetworkListener
	ServiceInterface
	laddr *net.UDPAddr
	raddr *net.UDPAddr
	conn  *net.UDPConn

	linkUp           bool
	interfaceAddress string
	ctx              *ServiceCtxt
}

func (s *AnnounceService) Init(c *ServiceCtxt) error {

	c.networkListener.Subscribe(s, "10.10.")
	log.Println("s init start", s.interfaceAddress)
	raddr, err := net.ResolveUDPAddr("udp", "239.0.0.72:16644")
	if err != nil {
		panic(err)
	}
	s.raddr = raddr
	s.ctx = c
	return nil
}
func (s *AnnounceService) Start(c *ServiceCtxt) error {

	return nil
}
func (s *AnnounceService) Stop(c *ServiceCtxt) error {
	return nil
}

func (s *AnnounceService) work() error {
	for {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		if s.conn != nil && s.ctx.listener != nil {
			res, err := s.conn.Write([]byte(hostname + "|" + s.ctx.listener.Addr().String()))
			if err != nil {
				log.Println("Announce connection loss. Exiting", res)
				break
			}
		} else {
			//log.Println("No mc connection")
		}
		time.Sleep(time.Second)
	}
	return nil
}
func (s *AnnounceService) unbindInterface() error {
	s.conn.Close()
	s.conn = nil
	return nil
}
func (s *AnnounceService) bindInterface(addr net.Addr) error {

	log.Println("bind", addr)
	laddr, err := net.ResolveUDPAddr("udp", strings.Split(addr.String(), "/")[0]+":0")
	if err != nil {
		return err
	}
	s.laddr = laddr
	s.linkUp = false
	s.interfaceAddress = addr.String()

	ucon, err := net.DialUDP("udp", s.laddr, s.raddr)
	if err != nil {
		return err
	}
	s.conn = ucon
	go s.work()
	return nil
}
func (s *AnnounceService) NetworkChanged(isUp bool, addr net.Addr) {
	if isUp {
		s.bindInterface(addr)
	} else {
		s.unbindInterface()
	}
}
