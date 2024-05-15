package service

/*
#cgo CFLAGS:-I/home/user/work/mini/src/work/tmp_rootfs/include -I/home/user/work/mini/src/work/tmp_rootfs/usr/include
#cgo LDFLAGS:-L /home/user/work/mini/src/work/tmp_rootfs/lib -L /home/user/work/mini/src/work/tmp_rootfs/usr/lib -lparted -lwim
#include <sys/stat.h>
#include <libudev.h>
#include <parted/parted.h>
#include <wimlib.h>

extern void NotifyCB(enum wimlib_progress_msg msg_type);

static inline enum wimlib_progress_status notify_cb(enum wimlib_progress_msg msg_type,
				  union wimlib_progress_info *info,
				  void *progctx){

					NotifyCB(msg_type);
					return WIMLIB_PROGRESS_STATUS_CONTINUE;
				  }

static inline int GO_wimlib_open_wim_with_progress(const wimlib_tchar *wim_file,
			      int open_flags,
			      WIMStruct **wim_ret,
			      void *progctx){
					return wimlib_open_wim_with_progress(wim_file, open_flags, wim_ret, notify_cb, progctx);
				  }
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"tr/com/havelsan/hloader/api"
	"unsafe"
)

//export NotifyCB
func NotifyCB(msgType C.enum_wimlib_progress_msg) {
	log.Println(msgType)
}

func (s *MaintenanceService) ApplyImage(req *api.ApplyImageRequest, server api.Maintain_ApplyImageServer) error {
	imgPath := req.GetImagePath()
	imgIndex := req.GetImageIndex()
	targetDisk := req.GetTargetDisk()
	targetPart := req.GetTargetPartition()
	log.Println("ApplyImage", imgPath, imgIndex, targetDisk, targetPart)
	cmd := exec.Command("/bin/wimapply", fmt.Sprintf("/nfs/%s", imgPath), imgIndex, "/dev/sda2")
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
func (s *MaintenanceService) ApplyImage2(req *api.ApplyImageRequest, server api.Maintain_ApplyImageServer) error {
	imgPath := req.GetImagePath()
	imgIndex := req.GetImageIndex()
	targetDisk := req.GetTargetDisk()
	targetPart := req.GetTargetPartition()
	log.Println("ApplyImage", imgPath, imgIndex, targetDisk, targetPart)
	var wim *C.WIMStruct = nil
	//cb := sys.Fu(func(msgType int, info *C.wimlib_progress_info, c uintptr) {
	//	return
	//})
	res := C.GO_wimlib_open_wim_with_progress(C.CString(fmt.Sprintf("/nfs/%s", imgPath)), 0, &wim, nil)
	if res != 0 {
		return errors.New("wim coudln't opened")
	} else {
		log.Println("Open wim", imgPath, ". OK")
		//res = C.wimlib_extract_image(wim, C.int(imgIndex), C.CString("/dev/sda2"), 0)
		log.Println("applywim result:", res)
		if res != 0 {
			C.wimlib_free(wim)
			return errors.New("couldn't apply wim image")
		}
		C.wimlib_free(wim)
	}
	return nil
}
func (s *MaintenanceService) FormatDisks(c context.Context, req *api.PartitionRequest) (*api.PartitionResponse, error) {
	log.Println(req)
	for _, disk := range req.GetDisks() {
		loc := disk.GetLocation()
		log.Println("DiskLoc:", loc)
		pDev := GetDiskByLocation(loc)
		if pDev == "" {
			return nil, errors.New("device not found")
		}

		dev := GetDevice(fmt.Sprintf("/dev/%s", pDev))
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
				pSize := pPart.pPart.geom.length*pPart.pPart.disk.dev.sector_size/1000/1000 + 1
				pType := C.ped_file_system_type_get(C.CString(part.GetType()))
				if pSize != C.longlong(part.Size) || pPart.pPart.fs_type != pType {
					log.Println("partition not match. Deleting...", pSize, pPart.pPart.fs_type, part.Size, pType)
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

type PedDisk struct {
	pDisk *C.PedDisk
}
type PedDevice struct {
	pDev *C.PedDevice
}
type PedPartition struct {
	pPart *C.PedPartition
}

func (p *PedDevice) GetDisk() *PedDisk {
	pDisk := C.ped_disk_new(p.pDev)
	if pDisk == nil {
		return nil
	}
	return &PedDisk{
		pDisk: pDisk,
	}
}
func (d *PedDevice) GetSysPath() string {
	return C.GoString(d.pDev.path)
}
func (p *PedDevice) MkLabel(diskType string) *PedDisk {
	pDiskType := C.ped_disk_type_get(C.CString(diskType))
	if pDiskType == nil {
		panic(fmt.Sprintf("Couldn't found disk type:%s", diskType))
	}
	pDisk := C.ped_disk_new_fresh(p.pDev, pDiskType)
	if pDisk == nil {
		panic("Couldn't create disk")
	}
	return &PedDisk{
		pDisk: pDisk,
	}
}
func (d *PedDisk) Commit() error {
	log.Println("Commiting changes")
	res := C.ped_disk_commit(d.pDisk)
	if res == 0 {
		return errors.New("couldn't commit")
	}
	log.Println("Commiting changes to disk")
	res = C.ped_disk_commit_to_dev(d.pDisk)
	if res == 0 {
		return errors.New("couldn't commit to dev")
	}
	log.Println("Commiting changes to os")
	res = C.ped_disk_commit_to_os(d.pDisk)
	if res == 0 {
		return errors.New("couldn't commit to os")
	}
	return nil
}
func (d *PedDisk) RmPart(partNum int) error {
	log.Println("RmPart", partNum)
	part := d.GetPartition(partNum)
	if part == nil {
		return errors.New("coudln't found partition to delete")
	}
	res := C.ped_disk_delete_partition(d.pDisk, part.pPart)
	if res == 0 {
		return errors.New("coudln't delete partition")
	}
	return nil
}
func (d *PedDisk) MkPart(partType string, size uint64, flags []string, label string) *PedPartition {
	log.Println("MKPart:", partType, size)
	pAlign := C.ped_disk_get_partition_alignment(d.pDisk)
	if pAlign == nil {
		panic("couldn't get alignment")
	}

	sectorSize := d.pDisk.dev.sector_size
	log.Println("SectorSize:", sectorSize)

	start := C.longlong(0)
	lastPartNum := d.GetPartitionCount()
	log.Println("PartCount:", lastPartNum)
	if lastPartNum > 0 {
		lastPart := d.GetPartition(lastPartNum)
		if lastPart != nil {
			start = lastPart.pPart.geom.end
			log.Println("LastPart End Offset:", start)
		}

	}
	end := start + C.longlong(size*1000*1000)/sectorSize
	pedFS := C.ped_file_system_type_get(C.CString(partType))
	if pedFS == nil {
		panic("Fs not found")
	}
	log.Println("Creating part", start, end)
	newPart := C.ped_partition_new(d.pDisk, C.PED_PARTITION_NORMAL, pedFS, start, end)
	if newPart == nil {
		panic("partition couldn't be created")
	}
	pedConstraint := C.ped_constraint_any(d.pDisk.dev)
	res := C.ped_disk_add_partition(d.pDisk, newPart, pedConstraint)
	if res != 0 {
		pPart := &PedPartition{
			pPart: newPart,
		}

		//modify flags
		for _, flag := range flags {
			pPart.SetFlag(flag)
		}

		//set label
		pPart.SetLabel(label)

		return pPart
	}
	log.Println("couldn't add partition to disk")
	return nil
}

func (d *PedDisk) UUID() []byte {
	uid := C.ped_disk_get_uuid(d.pDisk)
	if uid == nil {
		return nil
	}
	data := C.GoBytes(unsafe.Pointer(uid), 16)
	return data
}

func (d *PedDisk) GetPartitionCount() int {
	return int(C.ped_disk_get_primary_partition_count(d.pDisk))
}

func (d *PedDisk) GetPartition(partNum int) *PedPartition {
	pPart := C.ped_disk_get_partition(d.pDisk, C.int(partNum))
	if pPart == nil {
		return nil
	}
	return &PedPartition{
		pPart: pPart,
	}
}

func (p *PedPartition) SetFlag(flag string) error {
	pFlag := C.ped_partition_flag_get_by_name(C.CString(flag))
	if pFlag < 0 {
		return errors.New("Couldn't found flag")
	}
	res := C.ped_partition_set_flag(p.pPart, pFlag, 1)
	if res == 0 {
		return errors.New("Couldn't set flag")
	}
	return nil
}
func (p *PedPartition) ClearFlag(flag string) error {
	pFlag := C.ped_partition_flag_get_by_name(C.CString(flag))
	if pFlag < 0 {
		return errors.New("Couldn't found flag")
	}
	res := C.ped_partition_set_flag(p.pPart, pFlag, 0)
	if res == 0 {
		return errors.New("Couldn't set flag")
	}
	return nil
}
func (p *PedPartition) SetLabel(label string) error {
	res := C.ped_partition_set_name(p.pPart, C.CString(label))
	if res == 0 {
		return errors.New("Couldn't set name of partition")
	}
	return nil
}
func (p *PedPartition) GetSize() uint64 {
	return uint64(p.pPart.geom.length / p.pPart.disk.dev.sector_size)
}
func (p *PedPartition) EndOffset() uint64 {
	return uint64(p.pPart.geom.end / p.pPart.disk.dev.sector_size)
}
func (p *PedPartition) StartOffset() uint64 {
	return uint64(p.pPart.geom.start / p.pPart.disk.dev.sector_size)
}

func (p *PedPartition) UUID() []byte {
	uid := C.ped_partition_get_uuid(p.pPart)
	if uid == nil {
		return nil
	}
	data := C.GoBytes(unsafe.Pointer(uid), 16)
	return data
}

func GetDevice(dev string) *PedDevice {
	pDev := C.ped_device_get(C.CString(dev))
	if pDev == nil {
		return nil
	}
	return &PedDevice{
		pDev: pDev,
	}
}

func Devices() []string {
	C.ped_device_probe_all()
	var pDev *C.PedDevice = nil
	var devs []*C.PedDevice
	for i := 0; i < 5; i++ {
		next, err := C.ped_device_get_next(pDev)
		if err != nil {
			fmt.Println(err)
			break
		}
		if next != nil {
			//fmt.Println(i, next)
			devs = append(devs, next)
			pDev = next
		}
	}
	//fmt.Println(devs)
	return nil
}

func GetDiskAndPartNum(part string) (string, int) {
	dev := part
	if strings.Contains(part, "/") {
		parts := strings.Split(part, "/")
		dev = parts[len(parts)-1]
	}
	devPath, err := os.Readlink("/sys/class/block/" + dev)
	if err != nil {
		panic(err)
	}
	parts := strings.Split(devPath, "/")
	parentDev := parts[len(parts)-2]
	var stat, pstat C.struct_stat
	ret := C.fstatat(0, C.CString(part), &stat, 0)
	if ret != 0 {
		panic(fmt.Sprintf("Dev: %s not found", part))
	}
	ret = C.fstatat(0, C.CString("/dev/"+parentDev), &pstat, 0)
	if ret != 0 {
		panic(fmt.Sprintf("Dev: %s not found", part))
	}
	//fmt.Println(devPath, parentDev, stat.st_rdev-pstat.st_rdev)

	return "/dev/" + parentDev, int(stat.st_rdev - pstat.st_rdev)
}

// todo: find a better way
func GetDiskByLocation(loc string) string {
	direntry, er := os.ReadDir("/sys/class/block/")
	if er != nil {
		return ""
	}
	for _, a := range direntry {
		info, _ := os.Readlink(fmt.Sprintf("/sys/class/block/%s", a.Name()))
		if strings.Contains(info, loc) {
			ovs := strings.Split(info, "/block/")
			return ovs[len(ovs)-1]
		}
	}
	return ""
}
