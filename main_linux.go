package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
	"tr/com/havelsan/hloader/service"

	"github.com/sevlyar/go-daemon"
)

type program struct {
	LogFile *os.File
	svr     *service.ServiceCtxt
	ctx     context.Context
}

var (
	signal = flag.String("s", "", "send signal to daemon")
	prg    = service.DefaultServiceCtxt()
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
		Args:        []string{"[hvl-loader]"},
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

	prg.Init()

	go worker()

	err = daemon.ServeSignals()
	if err != nil {
		log.Printf("Error: %s", err.Error())
	}
	log.Println("daemon terminated")

	addr, err := net.ResolveUDPAddr("udp", "10.10.11.1:6644")
	if err != nil {
		log.Println("Can't resolve remote debug endpoint")
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Println("Can't dial remote debug endpoint")
		return
	}

	log.SetOutput(conn)

}

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func worker() {

	fmt.Println("ENV VARIABLES")
	for _, k := range os.Environ() {
		fmt.Printf("%s=%s", k, os.Getenv(k))
	}
	prg.Start()
	<-stop
	prg.Stop()

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
