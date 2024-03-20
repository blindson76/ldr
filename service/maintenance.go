package service

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"tr/com/havelsan/hloader/api"
)

type MaintenanceService struct {
	ServiceInterface
	api.MaintainServer
}

func (s *MaintenanceService) Init(c *ServiceCtxt) error {
	wd, _ := os.Executable()
	log.Println("Registering maintenance service", wd)
	api.RegisterMaintainServer(c.gs, s)
	return nil
}
func (s *MaintenanceService) Start(c *ServiceCtxt) error {
	return nil
}
func (s *MaintenanceService) Stop(c *ServiceCtxt) error {
	return nil
}

func (s *MaintenanceService) UpdateLoader(req api.Maintain_UpdateLoaderServer) error {
	log.Println("Update req", req)
	var file *os.File = nil
	var total uint32 = 0
	var received uint32 = 0
	var fname string
	hash := sha1.New()
	for {

		request, err := req.Recv()
		if err != nil {
			log.Println("recv err", err.Error())
			return err
		}
		switch u := request.Data.(type) {
		case *api.UploadRequest_Info:
			log.Println("Info:", u.Info)
			total = u.Info.Size
			fname = u.Info.GetName()
			// f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0644)
			f, err := ioutil.TempFile("", "hvl_*")
			if err != nil {
				log.Println("Open error", err.Error())
				return err
			}
			log.Println("File:", fname, f)
			defer f.Close()
			file = f
		case *api.UploadRequest_Chunk:
			// log.Println("Chunk", u.Chunk.Seq)
			n, err := file.Write(u.Chunk.Data)
			if err != nil {
				log.Println("Write error", err.Error())
				return err
			}
			received += uint32(n)
			// log.Println("Write bytes", n)
			req.Send(&api.UploadResponse{
				Data: &api.UploadResponse_Progress{
					Progress: strconv.FormatInt(int64(int(received)*100/int(total)), 10) + "%",
				},
			})
			if received == total {
				log.Println("Done")
			}
		case *api.UploadRequest_Hash:
			log.Println("Finish", u.Hash)
			ret, err := file.Seek(0, 0)
			if err != nil || ret != 0 {
				return errors.New("seek error:" + err.Error())
			}
			n, err := io.Copy(hash, file)
			if err != nil {
				return errors.New("copy error:" + err.Error())
			}
			if n != int64(total) {
				return errors.New("hashing error")
			}
			sum := hex.EncodeToString(hash.Sum(nil))
			if sum != u.Hash {
				log.Println("HASH Error", u.Hash, sum)
				return errors.New("hash error")
			}
			wd, err := os.Executable()
			if err != nil {
				return err
			}
			log.Println("FilePath:", file.Name(), wd)
			cmd := exec.Command("cmd", "/c", "start", "cmd", "/k", "sc stop hvl-loader & copy /y "+file.Name()+" "+wd+" & sc start hvl-loader & del /q /f "+file.Name())
			err = cmd.Run()
			if err != nil {
				log.Println("Run error", err.Error())
				return err
			}
			return nil
		}
	}

}
