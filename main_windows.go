package main

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"tr/com/havelsan/hloader/service"

	"github.com/judwhite/go-svc"
)

type program struct {
	LogFile *os.File
	svr     *service.ServiceCtxt
	ctx     context.Context
}

func (p *program) Context() context.Context {
	return p.ctx
}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prg := program{
		svr: service.DefaultServiceCtxt(),
		ctx: ctx,
	}

	defer func() {
		if prg.LogFile != nil {
			if closeErr := prg.LogFile.Close(); closeErr != nil {
				log.Printf("error closing '%s': %v\n", prg.LogFile.Name(), closeErr)
			}
		}
	}()

	// call svc.Run to start your program/service
	// svc.Run will call Init, Start, and Stop
	if err := svc.Run(&prg); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {

	// write to "example.log" when running as a Windows Service
	if env.IsWindowsService() {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return err
		}

		logPath := filepath.Join(dir, "loader.log")

		f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		p.LogFile = f

		log.SetOutput(f)
	} else {
		// return errors.New("this is not windows service")
	}

	addr, err := net.ResolveUDPAddr("udp", "10.10.11.1:6644")
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}

	log.SetOutput(conn)
	log.Println("starting loader service")
	err = p.svr.Init()
	return err
}

func (p *program) Start() error {
	log.Printf("Starting...\n")
	p.svr.Start()
	return nil
}

func (p *program) Stop() error {
	log.Printf("Stopping...\n")
	if err := p.svr.Stop(); err != nil {
		return err
	}
	log.Printf("Stopped.\n")
	return nil
}
