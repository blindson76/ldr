package parted

/*
#cgo CFLAGS:-I/home/user/work/mini/src/work/tmp_rootfs/include -I/home/user/work/mini/src/work/tmp_rootfs/usr/include
#cgo LDFLAGS:-L /home/user/work/mini/src/work/tmp_rootfs/lib -L /home/user/work/mini/src/work/tmp_rootfs/usr/lib -lparted
#include <sys/stat.h>
#include <sys/mount.h>
#include <mntent.h>
#include <libudev.h>
#include <parted/parted.h>

static inline char* GetMount(char* file){

	struct mntent *ent;
	FILE *aFile;

	aFile = setmntent("/proc/mounts", "r");
	if (aFile == NULL) {
		return NULL;
	}
	while (NULL != (ent = getmntent(aFile))) {
		if (strcmp(file, ent->mnt_fsname) == 0){
			return ent->mnt_dir;
		}
	}
	endmntent(aFile);
	return NULL;
}

*/
import "C"
import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"
)

type PedDisk struct {
	pDisk *C.PedDisk
	dev   *PedDevice
}
type PedDevice struct {
	pDev *C.PedDevice
}
type PedPartition struct {
	pPart *C.PedPartition
}

type PedAlignment struct {
	pAlign *C.PedAlignment
}

type PedConstraint struct {
	pConst *C.PedConstraint
}
type PedGeometry struct {
	pGeo *C.PedGeometry
}

func (p *PedDevice) GetDisk() *PedDisk {
	pDisk := C.ped_disk_new(p.pDev)
	if pDisk == nil {
		return nil
	}
	return &PedDisk{
		pDisk: pDisk,
		dev:   p,
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
		dev:   p,
	}
}
func (p *PedDevice) Constraint() *PedConstraint {
	pConst := C.ped_device_get_constraint(p.pDev)
	if pConst == nil {
		return nil
	}
	return &PedConstraint{
		pConst: pConst,
	}
}

// ped_device_get_minimal_aligned_constraint
func (p *PedDevice) ConstraintMinAlign() *PedConstraint {
	pConst := C.ped_device_get_minimal_aligned_constraint(p.pDev)
	if pConst == nil {
		return nil
	}
	return &PedConstraint{
		pConst: pConst,
	}
}
func (p *PedDevice) ConstraintOptAlign() *PedConstraint {
	pConst := C.ped_device_get_optimal_aligned_constraint(p.pDev)
	if pConst == nil {
		return nil
	}
	return &PedConstraint{
		pConst: pConst,
	}
}

func (p *PedDevice) AlignmentOpt() *PedAlignment {
	pAlign := C.ped_device_get_optimum_alignment(p.pDev)
	if pAlign == nil {
		return nil
	}
	return &PedAlignment{
		pAlign: pAlign,
	}
}
func (p *PedDevice) AlignmentMin() *PedAlignment {
	pAlign := C.ped_device_get_minimum_alignment(p.pDev)
	if pAlign == nil {
		return nil
	}
	return &PedAlignment{
		pAlign: pAlign,
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
	lcm := LCM(34, 2048)
	lstart := (((start - 1) / C.longlong(lcm)) + 1) * C.longlong(lcm)
	start = lstart
	log.Println(lstart)
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

	pPart := &PedPartition{
		pPart: newPart,
	}
	log.Printf("Created partition start:%d, end:%d, size:%d", pPart.StartOffset(), pPart.EndOffset(), pPart.GetSize())
	pedConstraint := d.dev.ConstraintOptAlign()

	res := C.ped_disk_add_partition(d.pDisk, newPart, pedConstraint.pConst)
	if res != 0 {
		pPart := &PedPartition{
			pPart: newPart,
		}
		log.Printf("Added   partition start:%d, end:%d, size:%d", pPart.StartOffset(), pPart.EndOffset(), pPart.GetSize())

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
func (d *PedDisk) GetAlignment() *PedAlignment {
	pAlign := C.ped_disk_get_partition_alignment(d.pDisk)
	if pAlign == nil {
		return nil
	}
	return &PedAlignment{
		pAlign: pAlign,
	}
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
		return errors.New("couldn't set flag")
	}
	return nil
}
func (p *PedPartition) ClearFlag(flag string) error {
	pFlag := C.ped_partition_flag_get_by_name(C.CString(flag))
	if pFlag < 0 {
		return errors.New("couldn't found flag")
	}
	res := C.ped_partition_set_flag(p.pPart, pFlag, 0)
	if res == 0 {
		return errors.New("couldn't set flag")
	}
	return nil
}
func (p *PedPartition) SetLabel(label string) error {
	res := C.ped_partition_set_name(p.pPart, C.CString(label))
	if res == 0 {
		return errors.New("couldn't set name of partition")
	}
	return nil
}
func (p *PedPartition) GetSize() uint64 {
	return uint64(p.pPart.geom.length)
}
func (p *PedPartition) GetSizeMB() uint64 {
	return uint64(p.pPart.geom.length*p.pPart.disk.dev.sector_size) / 1000 / 1000
}
func (p *PedPartition) EndOffset() uint64 {
	return uint64(p.pPart.geom.end)
}
func (p *PedPartition) StartOffset() uint64 {
	return uint64(p.pPart.geom.start)
}

func (p *PedPartition) GetFSType() string {
	return C.GoString(p.pPart.fs_type.name)
}
func (p *PedPartition) UUID() []byte {
	uid := C.ped_partition_get_uuid(p.pPart)
	if uid == nil {
		return nil
	}
	data := C.GoBytes(unsafe.Pointer(uid), 16)
	return data
}

func NewConstMin(geo *PedGeometry) *PedConstraint {
	pConst := C.ped_constraint_new_from_min(geo.pGeo)
	if pConst == nil {
		return nil
	}
	return &PedConstraint{
		pConst: pConst,
	}
}
func NewConstMax(geo *PedGeometry) *PedConstraint {
	pConst := C.ped_constraint_new_from_max(geo.pGeo)
	if pConst == nil {
		return nil
	}
	return &PedConstraint{
		pConst: pConst,
	}
}
func NewGeometry(dev *PedDevice, start, end int) *PedGeometry {
	pGeo := C.ped_geometry_new(dev.pDev, C.PedSector(start), C.PedSector(end))
	if pGeo == nil {
		return nil
	}
	return &PedGeometry{
		pGeo: pGeo,
	}
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

func Mkntfs(target, label string) error {

	cmd := exec.Command("/usr/sbin/mkntfs", "-Q", target)
	if label != "" {
		cmd = exec.Command("/usr/sbin/mkntfs", "-Q", "-L", label, target)
	}
	log.Writer()
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
func Mkvfat(target, label string) error {
	cmd := exec.Command("/sbin/mkfs.vfat", "-n", label, target)
	if label != "" {
		cmd = exec.Command("/sbin/mkfs.vfat", target)
	}
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
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
	fmt.Println(devs)
	return nil
}
func GetDevByPartNum(disk string, partNum int) string {
	devName := strings.Split(disk, "/dev/")[1]
	read, err := os.ReadFile("/proc/partitions")
	if err != nil {
		return ""
	}
	devices := strings.Split(string(read), "\n")[1:]
	for _, device := range devices {
		fields := strings.Fields(device)
		if len(fields) != 4 {
			continue
		}
		if fields[3] == devName {
			major := fields[0]
			minor := fields[1]
			minorNum, err := strconv.Atoi(minor)
			if err != nil {
				continue
			}
			minor = strconv.Itoa(minorNum + partNum)
			for _, device := range devices {
				fields := strings.Fields(device)
				if len(fields) != 4 {
					continue
				}
				if fields[0] == major && fields[1] == minor {
					return fmt.Sprintf("/dev/%s", fields[3])
				}

			}
		}
	}
	return ""
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

func CheckMount(target string) string {
	return C.GoString(C.GetMount(C.CString(target)))

}

// todo: find a better way
func GetDiskDevByLocation(loc string) string {
	direntry, er := os.ReadDir("/sys/class/block/")
	if er != nil {
		return ""
	}
	for _, a := range direntry {
		info, _ := os.Readlink(fmt.Sprintf("/sys/class/block/%s", a.Name()))
		if strings.Contains(info, loc) {
			ovs := strings.Split(info, "/block/")
			return fmt.Sprintf("/dev/%s", ovs[len(ovs)-1])
		}
	}
	return ""
}

func Mount(target, fsType string) (string, error) {
	dir, err := os.MkdirTemp("", "wim")
	if err != nil {
		return "", err
	}
	err = nil
	switch fsType {
	case "ntfs":
		_, err = exec.Command("/usr/bin/ntfs-3g", target, dir).CombinedOutput()
	case "fat32":
		_, err = exec.Command("/bin/mount", target, dir).CombinedOutput()
	case "winregfs":
		_, err = exec.Command("/bin/mount.winregfs", target, dir).CombinedOutput()

	}
	if err != nil {
		return "", err
	}
	return dir, nil
}
func Mount2(target string) (string, error) {
	dir, err := os.MkdirTemp("", "wim")
	if err != nil {
		return "", err
	}

	res, err := C.mount(C.CString(target), C.CString(dir), C.CString("fuseblk"), 0, unsafe.Pointer(C.CString("allow_other,blksize=4096,fd=4,ro")))
	if res != 0 {
		return "", err
	}
	return dir, nil
}
func Unmount(dir string) error {
	log.Println("unmounting", dir)
	ret := C.umount2(C.CString(dir), C.MNT_FORCE)
	if ret != 0 {
		return errors.New("could't umount")
	}
	log.Println("Deleting temporary")
	os.RemoveAll(dir)
	return nil
}
func BCDFix(espDisk string, espPartition uint32, osDisk string, osPartition uint32) error {
	log.Printf("BCDFix %s %d %s %d", espDisk, espPartition, osDisk, osPartition)
	bcdDiskDev := GetDiskDevByLocation(espDisk)
	if bcdDiskDev == "" {
		return fmt.Errorf("couldn't find esp disk:%s", espDisk)
	}
	log.Printf("Found bcdDevDisk:%s", bcdDiskDev)

	espPartDev := GetDevByPartNum(bcdDiskDev, int(espPartition))
	if espPartDev == "" {
		return fmt.Errorf("couldn't find esp partition:%s:%d", espDisk, espPartition)
	}
	log.Printf("Found espPartDev:%s", espPartDev)

	log.Printf("Mounting %s", espPartDev)
	mntDir, err := Mount(espPartDev, "fat32")
	if err != nil {
		return err
	}
	log.Printf("Mount %s to %s", espPartDev, mntDir)
	defer Unmount(mntDir)

	osDiskDev := GetDiskDevByLocation(osDisk)
	if osDiskDev == "" {
		return fmt.Errorf("couldn't find os disk:%s", osDisk)
	}
	osDev := GetDevice(osDiskDev)
	if osDev == nil {
		return fmt.Errorf("couldn't find esp disk dev:%s", espDisk)
	}
	pOsDisk := osDev.GetDisk()
	if pOsDisk == nil {
		return fmt.Errorf("couldn't find esp disk:%s", espDisk)
	}
	diskUUID := EncodeUUID(pOsDisk.UUID())
	pOSPart := pOsDisk.GetPartition(int(osPartition))
	if pOSPart == nil {
		return fmt.Errorf("couldn't find os part:%s:%d", osDisk, osPartition)
	}
	partUUID := EncodeUUID(pOSPart.UUID())
	log.Println(hex.EncodeToString(diskUUID), hex.EncodeToString(partUUID))

	bcdFilePath := fmt.Sprintf("%s/EFI/Microsoft/Boot/BCD", mntDir)
	log.Println(bcdFilePath)
	bcdFile, err := os.Stat(bcdFilePath)
	if err != nil {
		return err
	}
	log.Println(bcdFile)

	log.Printf("mounting %s", bcdFilePath)
	mntBCD, err := Mount(bcdFilePath, "winregfs")
	if err != nil {
		log.Println("Mount Failed", err)
		return err
	}
	log.Printf("Mount %s to %s", bcdFilePath, mntBCD)
	defer Unmount(mntBCD)

	err = filepath.Walk(mntBCD, func(dir string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() {
			fPath := path.Join(dir, "")
			if strings.Contains(fPath, "1000001/Element.bin") {
				log.Printf("BCD entry: %s", fPath)
				data, err := os.ReadFile(fPath)
				if err != nil {
					return fmt.Errorf("couldn't read bcd entry:%s", fPath)

				}
				log.Printf("O:%s", hex.EncodeToString(data))
				newData := data[:]
				copy(newData[32:], partUUID)
				copy(newData[56:], diskUUID)
				log.Printf("N:%s", hex.EncodeToString(newData))
				err = os.WriteFile(fPath, newData, 0777)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// greatest common divisor (GCD) via Euclidean algorithm
func GCD(a, b int) int {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

// find Least Common Multiple (LCM) via GCD
func LCM(a, b int, integers ...int) int {
	result := a * b / GCD(a, b)

	for i := 0; i < len(integers); i++ {
		result = LCM(result, integers[i])
	}

	return result
}
func EncodeUUID(data []byte) []byte {
	ndata := make([]byte, 16)
	copy(ndata, data)
	binary.LittleEndian.PutUint32(ndata[0:], binary.BigEndian.Uint32(data[0:]))
	binary.LittleEndian.PutUint16(ndata[4:], binary.BigEndian.Uint16(data[4:]))
	binary.LittleEndian.PutUint16(ndata[6:], binary.BigEndian.Uint16(data[6:]))
	return ndata
}
