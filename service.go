package main

import (
	"tr/com/havelsan/hloader/service"
)

type ServiceCtx struct {
	services []service.ServiceInterface
}

func (s *ServiceCtx) start() {
	for _, svc := range s.services {
		go svc.Start()
	}
}

func (s *ServiceCtx) stop() error {

	for _, svc := range s.services {
		svc.Stop()
	}
	return nil
}

func (s *ServiceCtx) init() error {
	for _, svc := range s.services {
		err := svc.Init()
		if err != nil {
			return err
		}
	}
	return nil
}
