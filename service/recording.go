package service

import (
	"context"
	"log"
	"sync"
	"tr/com/havelsan/hloader/api"
)

type RecordingService struct {
	ServiceInterface
	api.RecordingServer
}

var (
	mu           sync.Mutex
	status       string = "idle"
	recirod_time string = "0"
	pid          uint
)

func (s *RecordingService) Init(c *ServiceCtxt) error {
	api.RegisterRecordingServer(c.gs, s)
	return nil
}
func (s *RecordingService) Start(c *ServiceCtxt) error {
	return nil
}
func (s *RecordingService) Stop(c *ServiceCtxt) error {
	return nil
}

func (s *RecordingService) Status(c context.Context, e *api.Empty) (*api.RecordStatusResponse, error) {
	return &api.RecordStatusResponse{
		Status: status,
		Time:   recirod_time,
	}, nil
}
func (s *RecordingService) RecordControl(c context.Context, req *api.RecordRequest) (*api.RecordResponse, error) {

	var err error = nil
	switch req.Command {
	case "start":
		err = s.rec_start()
		break
	case "stop":
		log.Println("Stop recv")
		err = s.rec_stop()
	}
	return &api.RecordResponse{
		Status: status,
	}, err
}
