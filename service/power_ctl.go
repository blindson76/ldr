package service

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"tr/com/havelsan/hloader/api"
	"tr/com/havelsan/hloader/util"
	"unicode/utf16"

	"github.com/ecks/uefi/efi/efivario"
	"github.com/ecks/uefi/efi/efivars"
	"google.golang.org/grpc"
)

type PowerCtlInterface interface {
	Shutdown() error
	Restart() error
	Logout() error
	RestartTo(target int32) error
}
type PowerCtl struct {
	ServiceInterface
	api.LoaderServer
	PowerCtlInterface
	gs        *grpc.Server
	listener  net.Listener
	announcer net.UDPConn
	efiCtx    efivario.Context
}

func (s *PowerCtl) PowerCtl(ctx context.Context, req *api.PowerCtlOrder) (*api.Result, error) {
	log.Println("Order ", req.Order, req.Param)
	var err error = nil
	switch req.Order {
	case api.PowerStatusCommand_Restart:
		err = s.Restart()
	case api.PowerStatusCommand_Logoff:
		err = s.Logout()
	case api.PowerStatusCommand_Shutdown:
		err = s.Shutdown()
	case api.PowerStatusCommand_RestartTo:
		err = s.RestartTo(req.Param)
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

func (s *PowerCtl) RestartTo(target int32) error {
	log.Println("RestartTo", target)
	search := "Windows Boot Manager"
	if target == 2 {
		search = "Red Hat"
	}

	attr, boorOder, err := efivars.BootOrder.Get(s.efiCtx)
	if err != nil {
		return err
	}

	log.Println("Search", search)
	targetIndex := -1
	log.Println("Order:", boorOder)
	for i, boot := range boorOder {
		desc := getBootDesc(s.efiCtx, fmt.Sprintf("Boot%04d", boot))
		desc = strings.TrimSpace(desc)
		if strings.Contains(desc, search) {
			targetIndex = i
		}
		log.Println("boot:", boot, "Desc:", desc)

	}
	if targetIndex >= 0 {
		//efivars.BootOrder.Set()

		newOrder := []uint16{boorOder[targetIndex]}
		log.Println("First:", newOrder)

		newOrder = append(newOrder, boorOder[:targetIndex]...)
		log.Println("Second:", newOrder)

		newOrder = append(newOrder, boorOder[targetIndex+1:]...)
		out := make([]byte, len(newOrder)*2)
		for i, v := range newOrder {
			binary.LittleEndian.PutUint16(out[i*2:], v)
		}
		log.Println("Out", out)
		err := s.efiCtx.Set("BootOrder", efivars.GlobalVariable, attr, out)
		if err != nil {
			log.Println("Set BootOrder error", err.Error())
			return err
		} else {
			return s.Restart()
		}

	}
	return errors.New("boot entry not found")
}

func (s *PowerCtl) RestartToOnce(target int32) error {
	log.Println("RestartTo", target)
	search := "Windows Boot Manager"
	if target == 2 {
		search = "Red Hat"
	} else if target != 1 {
		return errors.New("invalid boot indx")
	}

	_, bootOder, err := efivars.BootOrder.Get(s.efiCtx)
	if err != nil {
		return err
	}

	log.Println("Search", search)
	targetIndex := -1
	log.Println("Order:", bootOder)
	for i, boot := range bootOder {
		desc := getBootDesc(s.efiCtx, fmt.Sprintf("Boot%04d", boot))
		desc = strings.TrimSpace(desc)
		if strings.Contains(desc, search) {
			targetIndex = i
		}
		log.Println("boot:", boot, "Desc:", desc)

	}
	if targetIndex >= 0 {

		_, next, _ := efivars.BootNext.Get(s.efiCtx)
		log.Println("Curr BootNex:", next)

		err = efivars.BootNext.Set(s.efiCtx, bootOder[targetIndex])

		_, next, _ = efivars.BootNext.Get(s.efiCtx)
		log.Println("New BootNex:", next)

		return err

	}
	return errors.New("boot entry not found")
}
func (s *PowerCtl) Init() error {
	err := s.LowInit()
	if err != nil {
		return err
	}
	s.efiCtx = efivario.NewDefaultContext()
	return nil
}

func getEnvVar(c efivario.Context, name string) []byte {
	size, err := c.GetSizeHint(name, efivars.GlobalVariable)
	if err != nil {
		return nil
	}
	out := make([]byte, size)
	_, _, err = c.Get(name, efivars.GlobalVariable, out)
	// log.Println(attr, sz, err, out)
	return out
}
func getBootDesc(c efivario.Context, name string) string {
	b := getEnvVar(c, name)
	// log.Println("Attrb", binary.LittleEndian.Uint32(b))
	// log.Println("FPathLen", binary.LittleEndian.Uint16(b[4:]))
	str := make([]uint16, len(b)/2)
	// log.Println("Len:", len(b))
	for i := 6; i < len(b)-2; i += 2 {
		v := binary.LittleEndian.Uint16(b[i:])
		if v == 0 {
			// log.Println("Ofset:", i)

			// str = str[:i]
			break
		}
		str[i/2-3] = v
	}

	return strings.TrimSpace(string(utf16.Decode(str)))

}
