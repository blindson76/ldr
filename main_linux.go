package main

import (
	"context"
	"flag"
	"log"
	"os"
	"syscall"

	"github.com/sevlyar/go-daemon"
)

type program struct {
	LogFile *os.File
	svr     *ServiceCtx
	ctx     context.Context
}

var (
	signal = flag.String("s", "", "send signal to daemon")
	prg    = DefaultService()
)

func (p *program) Context() context.Context {
	return p.ctx
}
func main() {
	flag.Parse()
	daemon.AddCommand(daemon.StringFlag(signal, "quit"), syscall.SIGQUIT, termHandler)
	daemon.AddCommand(daemon.StringFlag(signal, "stop"), syscall.SIGTERM, termHandler)
	daemon.AddCommand(daemon.StringFlag(signal, "reload"), syscall.SIGHUP, reloadHandler)

	cntxt := &daemon.Context{
		PidFileName: "/var/run/hvlloader.pid",
		PidFilePerm: 0644,
		LogFileName: "/var/log/hvlloader.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[go-daemon sample]"},
	}
	if len(daemon.ActiveFlags()) > 0 {
		d, err := cntxt.Search()
		if err != nil {
			log.Fatalf("Unable to send signal")
		}
		daemon.SendCommands(d)
		return
	}
	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatalln(err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()

	log.Println("--------------------------")
	log.Println("Daemon started")

	prg.init()

	go worker()

	err = daemon.ServeSignals()
	if err != nil {
		log.Printf("Error: %s", err.Error())
	}
	log.Println("daemon terminated")
}

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func worker() {
	prg.start()
	<-stop
	prg.stop()

	done <- struct{}{}
}
func termHandler(sig os.Signal) error {
	log.Println("terminating...")
	stop <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-done
	}
	return nil
}

func reloadHandler(sig os.Signal) error {
	log.Println("reload...")
	return nil
}