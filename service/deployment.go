package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"time"
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
func (s *DeploymentService) Upload(req api.Deployment_UploadServer) error {
	var file *os.File = nil
	var total uint32 = 0
	var received uint32 = 0
	var fname string
	var destination string
	hash := sha1.New()
	mu := sync.Mutex{}
	var stop chan int = make(chan int)
	for {

		request, err := req.Recv()
		//log.Println("upload request", req)
		if err != nil {
			log.Println("recv err", err.Error())
			return err
		}
		switch u := request.Data.(type) {
		case *api.UploadRequest_Info:
			//log.Println("Info1:", u.Info)
			total = u.Info.GetSize()
			fname = u.Info.GetName()
			destination = u.Info.GetDestination()
			dstPath := path.Join("c:/application", destination)
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				if err = os.MkdirAll(dstPath, 0700); err != nil {
					return err
				}
			}
			f, err := os.OpenFile(path.Join(dstPath, fname), os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Println("Open error", err.Error())
				return err
			}
			//log.Println("File123::", fname, f)
			defer f.Close()
			file = f
			if !mu.TryLock() {
				return errors.New("mutex lock error")
			}
			if total == 0 {
				mu.Unlock()
				return nil
			}
			go func() {
				for {
					select {
					case <-stop:
						req.Send(&api.UploadResponse{
							Data: &api.UploadResponse_Progress{
								Progress: strconv.FormatInt(int64(int(received)*100/int(total)), 10) + "%",
							},
						})
						return
					default:
						req.Send(&api.UploadResponse{
							Data: &api.UploadResponse_Progress{
								Progress: strconv.FormatInt(int64(int(received)*100/int(total)), 10) + "%",
							},
						})
						time.Sleep(time.Millisecond * 200)
					}
				}
			}()
		case *api.UploadRequest_Chunk:
			// log.Println("Chunk", u.Chunk.Seq)
			n, err := file.Write(u.Chunk.Data)
			if err != nil {
				log.Println("Write error", err.Error())
				return err
			}
			received += uint32(n)
			// log.Println("Write bytes", n)
			if received == total {
				//log.Println("Done!1")
				stop <- 0
				mu.Unlock()
			}
		case *api.UploadRequest_Hash:
			//log.Println("hash req")
			mu.Lock()
			defer mu.Unlock()
			//log.Println("Finish", u.Hash)
			ret, err := file.Seek(0, 0)
			if err != nil || ret != 0 {
				return errors.New("seek error:" + err.Error())
			}
			n, err := io.Copy(hash, file)
			if err != nil {
				return errors.New("copy error:" + err.Error())
			}
			if n != int64(total) {
				log.Println("hash error", n, total)
				return errors.New("hashing error")
			}
			sum := hex.EncodeToString(hash.Sum(nil))
			if sum != u.Hash {
				log.Println("HASH Error", u.Hash, sum)
				return errors.New("hash error")
			}
			return nil
		}
	}
}
