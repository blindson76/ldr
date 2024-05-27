package wim

/*
#cgo CFLAGS:-I/home/user/work/mini/src/work/tmp_rootfs/include -I/home/user/work/mini/src/work/tmp_rootfs/usr/include
#cgo LDFLAGS:-L /home/user/work/mini/src/work/tmp_rootfs/lib -L /home/user/work/mini/src/work/tmp_rootfs/usr/lib -lwim -lntfs-3g
#include <string.h>
#include <sys/stat.h>
#include <libudev.h>
#include <wimlib.h>
#include <ntfs-3g/volume.h>

extern void notifyCB(enum wimlib_progress_msg msg_type, union wimlib_progress_info *info, void *progctx);

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
	"fmt"
	"log"
	"runtime"
	"tr/com/havelsan/hloader/tools/parted"
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

func (w *WIM) Apply(index uint32, target, fsType string) (chan string, chan error) {
	w.chProgress = make(chan string)
	w.chCancel = make(chan int)
	w.chFinish = make(chan error)
	log.Println("Checking existing mounts")
	dir := parted.CheckMount(target)
	if dir != "" {
		log.Printf("Found mount:%s. Umounting...", dir)
		if err := parted.Unmount(dir); err != nil {
			log.Println("umount failed")
			w.chFinish <- err
			return nil, w.chFinish
		}
	}
	var err error = nil
	switch fsType {
	case "ntfs":
		err = parted.Mkntfs(target, "")
	case "fat32":
		err = parted.Mkvfat(target, "")
	}
	if err != nil {
		log.Printf("mkfs error for %s", fsType)
		w.chFinish <- err
		return nil, w.chFinish
	}
	log.Printf("mounting %s", target)
	dir, err = parted.Mount(target, fsType)
	if err != nil {
		w.chFinish <- err
		return nil, w.chFinish
	}
	log.Printf("mounted to %s", dir)
	//defer unmount(dir)

	log.Println("Applying image to", target, dir)
	go func() {
		res, err := C.wimlib_extract_image(w.wim, C.int(index), C.CString(dir), 0)
		log.Println("apply done", res)
		if res != 0 {
			w.chFinish <- err
		}
		defer close(w.chProgress)
		defer close(w.chFinish)

		defer parted.Unmount(dir)
		defer w.pin.Unpin()
	}()
	return w.chProgress, w.chFinish
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
		select {
		case w.chProgress <- status:
		default:
		}
		w.chFinish <- nil
		//defer close(w.chProgress)
		return
	}
	if status != "" {
		select {
		case w.chProgress <- status:
		default:
		}
	}
}
