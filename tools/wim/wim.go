package wim

/*
#cgo CFLAGS:-I/home/user/work/mini/src/work/tmp_rootfs/include -I/home/user/work/mini/src/work/tmp_rootfs/usr/include
#cgo LDFLAGS:-L /home/user/work/mini/src/work/tmp_rootfs/lib -L /home/user/work/mini/src/work/tmp_rootfs/usr/lib -lwim -lntfs-3g
#include <string.h>
#include <sys/stat.h>
#include <sys/mount.h>
#include <libudev.h>
#include <wimlib.h>
#include <ntfs-3g/volume.h>
#include <mntent.h>

extern void notifyCB(enum wimlib_progress_msg msg_type, union wimlib_progress_info *info, void *progctx);

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
static inline enum wimlib_progress_status notify_cb(enum wimlib_progress_msg msg_type,
				  union wimlib_progress_info *info,
				  void *progctx){
					notifyCB(msg_type, info, progctx);
					return WIMLIB_PROGRESS_STATUS_CONTINUE;
				  }

static inline int c_wimlib_open_wim_with_progress(const wimlib_tchar *wim_file,
			      int open_flags,
			      WIMStruct **wim_ret,
			      void *progctx){
					return wimlib_open_wim_with_progress(wim_file, open_flags, wim_ret, notify_cb, progctx);
				  }
*/
import "C"
import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"unsafe"
)

type WIMNotifyHandler interface {
	ExtractProgress(uint32)
}
type WIM struct {
	Image      string
	Handlers   WIMNotifyHandler
	chProgress chan string
	chCancel   chan int
	chFinish   chan error
	wim        *C.WIMStruct
	pin        *runtime.Pinner
}

func WIMOpen(image string) (*WIM, error) {
	w := &WIM{
		Image: image,
		pin:   &runtime.Pinner{},
	}
	err := w.Open()
	if err != nil {
		return nil, err
	}
	return w, nil
}
func (w *WIM) Open() error {
	pnt := unsafe.Pointer(&w)
	w.pin.Pin(&w.wim)
	w.pin.Pin(pnt)
	w.pin.Pin(w)
	w.pin.Pin(&w)
	res, err := C.c_wimlib_open_wim_with_progress(C.CString(w.Image), 0, &w.wim, pnt)
	if res != 0 {
		return err
	}
	return nil
}

func (w *WIM) Apply(index uint32, target string) (chan string, chan error) {
	w.chProgress = make(chan string)
	w.chCancel = make(chan int)
	w.chFinish = make(chan error)
	dir := checkMount(target)
	if dir != "" {
		unmount(dir)
	}
	err := mkntfs(target)
	if err != nil {
		w.chFinish <- err
		return nil, w.chFinish
	}
	dir, err = mount(target)
	if err != nil {
		log.Fatal(err)
	}
	//defer unmount(dir)

	log.Println("Applying image to", target, dir)
	go func() {
		res, err := C.wimlib_extract_image(w.wim, C.int(index), C.CString(dir), 0)
		log.Println("apply done", res, err)
		if res != 0 {
			w.chFinish <- err
		}
		defer close(w.chProgress)
		defer close(w.chFinish)

		defer unmount(dir)
		defer w.pin.Unpin()
	}()
	return w.chProgress, w.chFinish
}
func checkMount(target string) string {
	return C.GoString(C.GetMount(C.CString(target)))

}
func mkntfs(target string) error {
	cmd := exec.Command("/usr/sbin/mkntfs", "-Q", target)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
func mount(target string) (string, error) {
	dir, err := os.MkdirTemp("", "wim")
	if err != nil {
		return "", err
	}

	_, err = exec.Command("/usr/bin/ntfs-3g", target, dir).CombinedOutput()
	if err != nil {
		return "", err
	}
	return dir, nil
}
func mount2(target string) (string, error) {
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
func unmount(dir string) error {
	log.Println("unmounting", dir)
	ret := C.umount2(C.CString(dir), C.MNT_FORCE)
	if ret != 0 {
		return errors.New("could't umount")
	}
	log.Println("Deleting temporary")
	os.RemoveAll(dir)
	return nil
}
func (w *WIM) Cancel() error {
	return nil
}

//export notifyCB
func notifyCB(msgType C.enum_wimlib_progress_msg, info *C.union_wimlib_progress_info, ctx unsafe.Pointer) {
	var w *WIM = *(**WIM)(ctx)
	if ctx == nil {
		return
	}
	var status string = ""
	switch msgType {
	case 0:
		status = "Extracting started"
	case 3:
		status = "Creating directories"
	case 4:
		data := (*C.struct_wimlib_progress_info_extract)(unsafe.Pointer(&info[0]))
		status = fmt.Sprintf("Extracting files (%d/%d %d%%)", data.completed_streams, data.total_streams, (data.completed_streams*C.ulong(100))/data.total_streams)
	case 6:
		status = "Extracting metadata"
	case 7:
		status = "Image applied successfully"
		defer close(w.chProgress)
	}
	if status != "" {
		log.Println(status)
		select {
		case w.chProgress <- status:
		default:
		}
	}
}

func WimTest() {

}
