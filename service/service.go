package service

import (
	"log"
	"net"
	"strings"
	"time"
	"tr/com/havelsan/hloader/util"

	"google.golang.org/grpc"
)

type ServiceInterface interface {
	Init(*ServiceCtxt) error
	Start(*ServiceCtxt) error
	Stop(*ServiceCtxt) error
}

type ServiceCtxt struct {
	util.NetworkListener
	services         []ServiceInterface
	gs               *grpc.Server
	laddr            net.Addr
	listener         net.Listener
	iface            *net.Interface
	linkUp           bool
	interfaceAddress string
	networkListener  *util.NetworkChangeNotifier
}

func DefaultServiceCtxt() *ServiceCtxt {
	return &ServiceCtxt{
		services: []ServiceInterface{
			&AnnounceService{
				interfaceAddress: "10.10.",
			},
			&PowerCtl{},
			&DeploymentService{},
			&RecordingService{},
			&MaintenanceService{},
		},
		interfaceAddress: "10.10",
		networkListener:  util.NewNetworkChangeNotifier(),
	}
}

func (s *ServiceCtxt) Init() error {
	s.networkListener.Subscribe(s, "10.10.")

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
	s.networkListener.Start()
	for _, svc := range s.services {
		svc.Start(s)
	}

	//return nil

}
func (s *ServiceCtxt) setLinkState(state bool, addr net.Addr) {
	if s.linkUp != state {
		s.linkUp = state
		if s.linkUp {
			log.Println("grpc UP")
			s.bindInterface(addr)

		} else {
			log.Println("grpc DOWN")
		}
	}
}
func (s *ServiceCtxt) Stop() error {

	for _, svc := range s.services {
		svc.Stop(s)
	}
	return nil
}

func (s *ServiceCtxt) work(listener net.Listener) {
	if err := s.gs.Serve(listener); err != nil {
		log.Printf("serve error")
	}

}

func (s *ServiceCtxt) bindInterface(addr net.Addr) {
	log.Println("grpc bind interfaces", addr)
	//time.Sleep(time.Second * 1)
	for {
		listener, err := net.Listen("tcp", strings.Split(addr.String(), "/")[0]+":0")
		if err != nil {
			log.Println("listen err", addr, err.Error())
			time.Sleep(time.Second * 2)
			continue
		}
		log.Println("grpc listener:", listener.Addr().String())
		s.listener = listener
		go s.work(listener)
		return
	}

}
func (s *ServiceCtxt) unbindInterface() {
	s.gs.Stop()
	s.listener.Close()
	s.listener = nil
	log.Println("grpc unbind interfaces")
}
func (s *ServiceCtxt) NetworkChanged(up bool, addr net.Addr) {
	log.Println("Network changed ", up)
	if up {
		s.bindInterface(addr)
	} else {
		s.unbindInterface()
	}
}
