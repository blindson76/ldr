package main

import (
	"log"
	"tr/com/havelsan/hloader/tools/parted"
	"tr/com/havelsan/hloader/tools/wim"
)

type WimHandler struct {
	wim.WIMNotifyHandler
}

func (w *WimHandler) ExtractProgress(progress uint32) {

}
func main() {
	disk := parted.GetDiskDevByLocation("pci0000:00/0000:00:0d.0")
	part := parted.GetDevByPartNum(disk, 4)
	log.Println(part)
	w, err := wim.WIMOpen("/media/sf_work/pxe/root/winpe.wim")
	if err != nil {
		panic(err)
	}
	log.Println("Open")
	prog, errch := w.Apply(1, part)

out:
	for {
		select {
		case res := <-prog:
			log.Println(res)
		case err := <-errch:
			log.Println(err)
			break out
		}
	}
	log.Println("finito")
}
