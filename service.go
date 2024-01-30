package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"tr/com/havelsan/hloader/api"
	"tr/com/havelsan/hloader/service"
	"tr/com/havelsan/hloader/util"

	"google.golang.org/grpc"
)

type ApiServer struct {
	api.LoaderServer
	svc service.PowerCtl
}

func (s *ApiServer) PowerCtl(ctx context.Context, req *api.PowerCtlOrder) (*api.Result, error) {
	log.Println("Order ", req.Order)
	switch req.Order {
	case api.PowerStatusCommand_Restart:
		err := s.svc.Restart()
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
	case api.PowerStatusCommand_Logoff:
		err := s.svc.Logout()
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
	case api.PowerStatusCommand_Shutdown:
		err := s.svc.Shutdown()
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
	return &api.Result{
		Result:  1,
		Message: "Invalid order",
	}, nil
}

type Service struct {
	gs        *grpc.Server
	listener  net.Listener
	announcer net.UDPConn
}

func (s *Service) start() {
	go s.work()
	log.Printf("Service startedsss")
}

func (s *Service) stop() error {
	s.gs.Stop()
	return nil
}

func (s *Service) work() {

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
		return
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
	api.RegisterLoaderServer(s.gs, &ApiServer{
		svc: &service.PowerCtlImp{},
	})

	go s.startAnnounce()
	if err := s.gs.Serve(listener); err != nil {
		log.Printf("serve error")
		return
	}
}

func (s *Service) startAnnounce() {
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
