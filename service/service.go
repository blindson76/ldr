package service

import (
	"errors"
	"log"
	"net"
	"strings"
	"tr/com/havelsan/hloader/util"

	"google.golang.org/grpc"
)

type ServiceInterface interface {
	Init(*ServiceCtxt) error
	Start(*ServiceCtxt) error
	Stop(*ServiceCtxt) error
}

type ServiceCtxt struct {
	services []ServiceInterface
	gs       *grpc.Server
	laddr    net.Addr
	listener net.Listener
	iface    *net.Interface
}

func DefaultServiceCtxt() *ServiceCtxt {
	return &ServiceCtxt{
		services: []ServiceInterface{
			&AnnounceService{},
			&PowerCtl{},
		},
	}
}

func (s *ServiceCtxt) Init() error {

	iface, addr := util.InterfaceByAddress("10.10.")
	if iface == nil {
		panic(errors.New("iface not found"))
	}
	s.iface = iface
	s.laddr = addr
	s.gs = grpc.NewServer()
	for _, svc := range s.services {
		err := svc.Init(s)
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *ServiceCtxt) Start() {

	listener, err := net.Listen("tcp", strings.Split(s.laddr.String(), "/")[0]+":0")
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(listener.Addr().String())
	s.listener = listener

	for _, svc := range s.services {
		svc.Start(s)
	}

	if err := s.gs.Serve(listener); err != nil {
		log.Printf("serve error")
	}
}

func (s *ServiceCtxt) Stop() error {

	for _, svc := range s.services {
		svc.Stop(s)
	}
	return nil
}
