package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os/exec"
	"strings"
	"tr/com/havelsan/hloader/api"
	"tr/com/havelsan/hloader/tools/parted"
	"tr/com/havelsan/hloader/tools/wim"
)

func (s *MaintenanceService) BCDFix(c context.Context, req *api.BCDFixRequest) (*api.BCDFixResponse, error) {
	espDisk := req.GetEspDisk()
	espPartition := req.EspPartition
	osDisk := req.GetOsDisk()
	osPart := req.GetOsPartition()

	err := parted.BCDFix(espDisk, espPartition, osDisk, osPart)
	if err != nil {
		return &api.BCDFixResponse{
			Status: err.Error(),
		}, err
	}
	return &api.BCDFixResponse{
		Status: "success",
	}, nil
}
func (s *MaintenanceService) ApplyImage(req *api.ApplyImageRequest, server api.Maintain_ApplyImageServer) error {
	imgPath := req.GetImagePath()
	imgIndex := req.GetImageIndex()
	targetDisk := req.GetTargetDisk()
	targetPart := req.GetTargetPartition()

	disk := parted.GetDiskDevByLocation(targetDisk)
	if disk == "" {
		return fmt.Errorf("couldn't find disk device:%s", targetDisk)
	}
	part := parted.GetDevByPartNum(disk, int(targetPart))
	if part == "" {
		return fmt.Errorf("couldn't find partition device:%s", targetPart)
	}
	dev := parted.GetDevice(disk)
	if dev == nil {
		return fmt.Errorf("couldn't find device:%s", disk)
	}
	ddisk := dev.GetDisk()
	if ddisk == nil {
		return fmt.Errorf("couldn't find disk:%s", disk)
	}
	pPart := ddisk.GetPartition(int(targetPart))
	if pPart == nil {
		return fmt.Errorf("couldn't find partition:%d", targetPart)
	}
	log.Println("Applying image for fs", pPart.GetFSType())
	if pPart.GetFSType() != "ntfs" {
		return s.ApplyImage2(req, server)
	}
	cmd := exec.Command("wimapply", fmt.Sprintf("/nfs/%s", imgPath), fmt.Sprintf("%d", imgIndex), fmt.Sprintf("%s", part))
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	for {
		tmp := make([]byte, 1024)
		n, err := stdout.Read(tmp)
		if err != nil {
			break
		}
		server.Send(&api.AplyImageStatus{
			Status: strings.TrimSpace(string(tmp[:n])),
		})
	}
	err = cmd.Wait()
	return err

}
func (s *MaintenanceService) ApplyImage2(req *api.ApplyImageRequest, server api.Maintain_ApplyImageServer) error {
	imgPath := req.GetImagePath()
	imgIndex := req.GetImageIndex()
	targetDisk := req.GetTargetDisk()
	targetPart := req.GetTargetPartition()

	disk := parted.GetDiskDevByLocation(targetDisk)
	part := parted.GetDevByPartNum(disk, int(targetPart))
	dev := parted.GetDevice(disk)
	if dev == nil {
		return fmt.Errorf("couldn't find device:%s", disk)
	}
	ddisk := dev.GetDisk()
	if ddisk == nil {
		return fmt.Errorf("couldn't find disk:%s", disk)
	}
	pPart := ddisk.GetPartition(int(targetPart))
	if pPart == nil {
		return fmt.Errorf("couldn't find partition:%d", targetPart)
	}
	w, err := wim.WIMOpen(fmt.Sprintf("/nfs/%s", imgPath))
	if err != nil {
		return err
	}
	server.Send(&api.AplyImageStatus{
		Status: "Applying image",
	})
	log.Println("Open")
	chProg, chErr := w.Apply(imgIndex, part, pPart.GetFSType())

	for {
		select {
		case res := <-chProg:
			log.Println(res)
			server.Send(&api.AplyImageStatus{
				Status: res,
			})
		case err := <-chErr:
			log.Println(err)
			if err != nil {
				return err
			}
			return nil
		}
	}

}

func (s *MaintenanceService) FormatDisks(req *api.PartitionRequest, server api.Maintain_FormatDisksServer) error {
	log.Println(req)
	for _, disk := range req.GetDisks() {
		loc := disk.GetLocation()
		log.Println("DiskLoc:", loc)
		diskDev := parted.GetDiskDevByLocation(loc)
		if diskDev == "" {
			return errors.New("device not found")
		}

		dev := parted.GetDevice(diskDev)
		if dev == nil {
			return errors.New("disk not found")
		}
		server.Send(&api.PartitionResponse{
			Status: fmt.Sprintf("Found device %s as %s", loc, diskDev),
		})
		pDisk := dev.GetDisk()
		if pDisk == nil {
			//create disk
			server.Send(&api.PartitionResponse{
				Status: fmt.Sprintf("No partition table found. Creating %s partition table", disk.PartitionType),
			})
			pDisk = dev.MkLabel(disk.PartitionType)
			if pDisk == nil {
				return errors.New("couldn't create disk")
			}

		}
		for i, part := range disk.GetPartitions() {
			pPart := pDisk.GetPartition(i + 1)
			if pPart == nil {
				//create part

				server.Send(&api.PartitionResponse{
					Status: fmt.Sprintf("Creating %dMB %s partition on disk %s", part.GetSize(), part.GetType(), diskDev),
				})
				pPart = pDisk.MkPart(part.GetType(), part.GetSize(), part.GetFlags(), part.GetLabel())
				if pPart == nil {
					return errors.New("couldn't create partition")
				}

			} else {
				pSize := pPart.GetSizeMB()
				pType := pPart.GetFSType()
				diff := float64(pSize) - float64(part.Size)
				if math.Abs(diff) > 2.0 || part.GetType() != pType {
					log.Println("partition not match. Deleting...", pSize, pType, part.Size, part.GetType())
					server.Send(&api.PartitionResponse{
						Status: fmt.Sprintf("Deleting existing partition %d on disk %s", i+1, diskDev),
					})
					for pi := i; i < pDisk.GetPartitionCount(); i++ {
						log.Println("Deleting partition", i+1)
						err := pDisk.RmPart(pi + 1)
						if err != nil {
							return err
						}
					}
					//create part
					pPart = pDisk.MkPart(part.GetType(), part.GetSize(), part.GetFlags(), part.GetLabel())
					if pPart == nil {
						return errors.New("couldn't create partition")
					}
				}

			}
		}
		if err := pDisk.Commit(); err != nil {
			return err
		}
	}

	//format partitions
	log.Println("Formatting...")
	server.Send(&api.PartitionResponse{
		Status: "Creating filesystems",
	})
	for _, disk := range req.GetDisks() {
		loc := disk.GetLocation()
		diskDev := parted.GetDiskDevByLocation(loc)
		if diskDev == "" {
			return errors.New("device not found")
		}

		dev := parted.GetDevice(diskDev)
		if dev == nil {
			return errors.New("disk not found")
		}
		pDisk := dev.GetDisk()
		for i, part := range disk.GetPartitions() {
			pPart := pDisk.GetPartition(i + 1)
			if pPart != nil {
				log.Printf("Formatting part:%d", i+1)
				devPart := parted.GetDevByPartNum(diskDev, i+1)
				if devPart == "" {
					return fmt.Errorf("couldn't find partition to format: %s:%d", diskDev, i+1)
				}
				server.Send(&api.PartitionResponse{
					Status: fmt.Sprintf("Creating %s fs on partition %s", part.GetType(), devPart),
				})
				var err error = nil
				switch part.GetType() {
				case "ntfs":
					err = parted.Mkntfs(devPart, part.GetLabel())
				case "fat32":
					err = parted.Mkvfat(devPart, part.GetLabel())
				}
				if err != nil {
					return err
				} else {
					log.Println("Format done")
				}

			}
		}
		if err := pDisk.Commit(); err != nil {
			return err
		}
	}
	return nil
}
