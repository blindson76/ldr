package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"tr/com/havelsan/hloader/api"
	"tr/com/havelsan/hloader/tools/parted"
	"tr/com/havelsan/hloader/tools/wim"
)

func (s *MaintenanceService) ApplyImage2(req *api.ApplyImageRequest, server api.Maintain_ApplyImageServer) error {
	wim.WimTest()
	imgPath := req.GetImagePath()
	imgIndex := req.GetImageIndex()
	targetDisk := req.GetTargetDisk()
	targetPart := req.GetTargetPartition()
	log.Println("ApplyImage", imgPath, imgIndex, targetDisk, targetPart)
	cmd := exec.Command("/bin/wimapply", fmt.Sprintf("/nfs/%s", imgPath), "1", "/dev/sda2")
	rd, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	for {
		tmp := make([]byte, 1024)
		_, err := rd.Read(tmp)
		log.Println(string(tmp))
		server.Send(&api.AplyImageStatus{
			Status: string(tmp),
		})
		if err != nil {
			break
		}
	}
	return nil
}

func (s *MaintenanceService) ApplyImage(req *api.ApplyImageRequest, server api.Maintain_ApplyImageServer) error {
	imgPath := req.GetImagePath()
	imgIndex := req.GetImageIndex()
	targetDisk := req.GetTargetDisk()
	targetPart := req.GetTargetPartition()

	disk := parted.GetDiskDevByLocation(targetDisk)
	part := parted.GetDevByPartNum(disk, int(targetPart))
	w, err := wim.WIMOpen(fmt.Sprintf("/nfs/%s", imgPath))
	if err != nil {
		panic(err)
	}
	log.Println("Open")
	chProg, chErr := w.Apply(imgIndex, part)

	for {
		select {
		case res := <-chProg:
			log.Println(res)
			server.Send(&api.AplyImageStatus{
				Status: res,
			})
		case err := <-chErr:
			log.Println(err)
			return err
		}
	}
}

func (s *MaintenanceService) FormatDisks(c context.Context, req *api.PartitionRequest) (*api.PartitionResponse, error) {
	log.Println(req)
	for _, disk := range req.GetDisks() {
		loc := disk.GetLocation()
		log.Println("DiskLoc:", loc)
		diskDev := parted.GetDiskDevByLocation(loc)
		if diskDev == "" {
			return nil, errors.New("device not found")
		}

		dev := parted.GetDevice(diskDev)
		if dev == nil {
			return nil, errors.New("disk not found")
		}
		pDisk := dev.GetDisk()
		if pDisk == nil {
			//create disk
			pDisk = dev.MkLabel(disk.PartitionType)
			if pDisk == nil {
				return nil, errors.New("couldn't create disk")
			}

		}
		for i, part := range disk.GetPartitions() {
			pPart := pDisk.GetPartition(i + 1)
			if pPart == nil {
				//create part
				pPart = pDisk.MkPart(part.GetType(), part.GetSize(), part.GetFlags(), part.GetLabel())
				if pPart == nil {
					return nil, errors.New("couldn't create partition")
				}

			} else {
				pSize := pPart.GetSizeMB() + 1
				pType := pPart.GetFSType()
				if pSize != part.Size || part.GetType() != pType {
					log.Println("partition not match. Deleting...", pSize, pType, part.Size, part.GetType())
					for pi := i; i < pDisk.GetPartitionCount(); i++ {
						err := pDisk.RmPart(pi + 1)
						if err != nil {
							return nil, err
						}
					}
					//create part
					pPart = pDisk.MkPart(part.GetType(), part.GetSize(), part.GetFlags(), part.GetLabel())
					if pPart == nil {
						return nil, errors.New("couldn't create partition")
					}
				}

			}
		}
		if err := pDisk.Commit(); err != nil {
			return nil, err
		}
	}
	return &api.PartitionResponse{Result: 0}, nil
}
