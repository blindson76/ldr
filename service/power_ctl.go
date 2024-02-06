package service

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"tr/com/havelsan/hloader/api"
	"tr/com/havelsan/hloader/util"

	"google.golang.org/grpc"
)

type PowerCtlInterface interface {
	Shutdown() error
	Restart() error
	Logout() error
}
type PowerCtl struct {
	ServiceInterface
	api.LoaderServer
	PowerCtlInterface
	gs        *grpc.Server
	listener  net.Listener
	announcer net.UDPConn
}

func (s *PowerCtl) PowerCtl(ctx context.Context, req *api.PowerCtlOrder) (*api.Result, error) {
	log.Println("Order ", req.Order)
	var err error = nil
	switch req.Order {
	case api.PowerStatusCommand_Restart:
		err = s.Restart()
	case api.PowerStatusCommand_Logoff:
		err = s.Logout()
	case api.PowerStatusCommand_Shutdown:
		err = s.Shutdown()
	}

	if err != nil {
		return &api.Result{
			Result:  -1,
			Message: err.Error(),
		}, nil
	} else {
		return &api.Result{
			Result:  0,
			Message: "Success",
		}, nil
	}
}

func (s *PowerCtl) Start() error {
	iface, addr := util.InterfaceByAddress("10.10.")
	if iface == nil {
		panic(errors.New("iface not found"))
	}
	laddr, err := net.ResolveUDPAddr("udp", strings.Split(addr.String(), "/")[0]+":0")
	if err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", strings.Split(addr.String(), "/")[0]+":0")
	if err != nil {
		log.Fatal(err.Error())
		return nil
	}
	log.Println(listener.Addr().String())

	raddr, err := net.ResolveUDPAddr("udp", "239.0.0.72:16644")
	if err != nil {
		panic(err)
	}
	ucon, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		panic(err)
	}
	s.announcer = *ucon
	s.listener = listener
	s.gs = grpc.NewServer()
	api.RegisterLoaderServer(s.gs, s)

	go s.startAnnounce()
	if err := s.gs.Serve(listener); err != nil {
		log.Printf("serve error")
		return nil
	}
	return nil
}
func (s *PowerCtl) startAnnounce() {
	for {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		res, err := s.announcer.Write([]byte(hostname + "|" + s.listener.Addr().String()))
		if err != nil {
			log.Println("Exiting", res)
			return
		}
		time.Sleep(time.Second)
	}
}
