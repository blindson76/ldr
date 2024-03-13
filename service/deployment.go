package service

import (
	"context"
	"log"
	"os"
	"os/exec"
	"tr/com/havelsan/hloader/api"
)

type DeploymentService struct {
	ServiceInterface
	api.DeploymentServer
}

func (s *DeploymentService) Init(c *ServiceCtxt) error {
	log.Println("Registering deployment service")
	api.RegisterDeploymentServer(c.gs, s)
	return nil
}
func (s *DeploymentService) Start(c *ServiceCtxt) error {
	return nil
}
func (s *DeploymentService) Stop(c *ServiceCtxt) error {
	return nil
}

func (s *DeploymentService) Info(c context.Context, req *api.InfoRequest) (*api.InfoResponse, error) {
	log.Println("Dep Info", req)
	hostname, err := os.Hostname()
	if err != nil {
		return &api.InfoResponse{
			NodeId:   5,
			Hostname: "os.Hostname()",
		}, nil
	} else {
		return &api.InfoResponse{
			NodeId:   5,
			Hostname: hostname,
		}, nil
	}
}
func (s *DeploymentService) Exec(c context.Context, req *api.ExecRequest) (*api.ExecResponse, error) {
	log.Println("Dep Exec", req)
	out, err := exec.Command(req.Proc, req.Args...).Output()
	if err != nil {
		errStr := err.Error()
		return &api.ExecResponse{
			Status: -1,
			Err:    &errStr,
		}, nil
	} else {
		outStr := string(out)
		return &api.ExecResponse{
			Status:   0,
			ExitCode: 0,
			Out:      &outStr,
		}, nil
	}
}
