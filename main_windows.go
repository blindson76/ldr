package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/judwhite/go-svc"
)

type program struct {
	LogFile *os.File
	svr     *ServiceCtx
	ctx     context.Context
}

func (p *program) Context() context.Context {
	return p.ctx
}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prg := program{
		svr: DefaultService(),
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
	err := p.svr.init()
	return err
}

func (p *program) Start() error {
	log.Printf("Starting...\n")
	p.svr.start()
	return nil
}

func (p *program) Stop() error {
	log.Printf("Stopping...\n")
	if err := p.svr.stop(); err != nil {
		return err
	}
	log.Printf("Stopped.\n")
	return nil
}