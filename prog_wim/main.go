package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/blindson76/uefi/efi/efivario"
	"github.com/blindson76/uefi/efi/efivars"
	"github.com/google/uuid"
	"golang.org/x/sys/windows"
)

func main() {
	err := Init()

	if err != nil {
		return
	}

	c := efivario.NewDefaultContext()
	_, order, err := efivars.BootOrder.Get(c)
	if err != nil {
		return
	}
	for _, bootIndex := range order {
		_, bootEntry, err := efivars.Boot(bootIndex).Get(c)
		if err != nil {
			continue
		}
		fmt.Println(bootEntry.DescriptionString())
		for _, devPath := range bootEntry.FilePathList {
			switch dp := devPath.(type) {
			default:
				fmt.Println(dp.Text())
			}
		}
		fmt.Println()
	}
	if true {
		return
	}
	BootEntries()
}
func Init() error {
	var token windows.Token
	tkp := windows.Tokenprivileges{}
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token)
	if err != nil {
		return err
	}
	//SeSystemEnvironmentPrivilege
	priv, err := windows.UTF16PtrFromString("SeShutdownPrivilege")
	if err != nil {
		return err
	}
	err = windows.LookupPrivilegeValue(nil, priv, &tkp.Privileges[0].Luid)
	if err != nil {
		return err
	}
	tkp.PrivilegeCount = 1
	tkp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	err = windows.AdjustTokenPrivileges(token, false, &tkp, 0, nil, nil)
	if err != nil {
		return err
	}
	priv, err = windows.UTF16PtrFromString("SeSystemEnvironmentPrivilege")
	if err != nil {
		return err
	}
	err = windows.LookupPrivilegeValue(nil, priv, &tkp.Privileges[0].Luid)
	if err != nil {
		return err
	}
	tkp.PrivilegeCount = 1
	tkp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	err = windows.AdjustTokenPrivileges(token, false, &tkp, 0, nil, nil)
	if err == nil {
		// log.Println("GOT ENVVAR privilege")
	}
	return nil
}
func BootEntries() []string {
	efiCtx := efivario.NewDefaultContext()
	_, boorOder, err := efivars.BootOrder.Get(efiCtx)
	if err != nil {
		return nil
	}
	entries := make([]string, len(boorOder))
	for i, boot := range boorOder {
		desc := getBootDesc(efiCtx, fmt.Sprintf("Boot%04d", boot))
		desc = strings.TrimSpace(desc)
		// log.Println("boot:", boot, "Desc:", desc)
		entries[i] = desc
	}
	return entries
}

// internal
func getEnvVar(c efivario.Context, name string) []byte {
	size, err := c.GetSizeHint(name, efivars.GlobalVariable)
	if err != nil {
		return nil
	}
	out := make([]byte, size)
	_, _, err = c.Get(name, efivars.GlobalVariable, out)
	// log.Println(attr, sz, err, out)
	return out
}
func getBootDesc(c efivario.Context, name string) string {
	b := getEnvVar(c, name)
	if len(b) == 0 {
		return ""
	}
	// log.Println("Boot Entry:", hex.EncodeToString(b))
	// log.Println(hex.EncodeToString(b))
	Attrib := binary.LittleEndian.Uint32(b)
	PathLen := binary.LittleEndian.Uint16(b[4:])
	fmt.Println("#", name)
	fmt.Println("Attrib:", Attrib)
	fmt.Println("PathLen:", PathLen)
	str := make([]uint16, len(b)/2)
	var desc string
	start := 0
	for i := 6; i < len(b)-2; i += 2 {
		v := binary.LittleEndian.Uint16(b[i:])
		if v == 0 {
			desc = string(utf16.Decode(str[:i/2-3]))
			start = i + 2
			break
		}
		str[i/2-3] = v
	}
	fmt.Println("Desc:", desc)

	// fp := efiv.NewFilePath(b[start : start+int(PathLen)])
	// fmt.Println(fp)
	// if true {
	// 	return ""
	// }
	ParsePath(b[start:start+int(PathLen)], 0)
	opts := b[start+int(PathLen):]
	if len(opts) > 0 {
		fmt.Println("Option:", hex.EncodeToString(opts))
	}

	return desc

}

func ParsePath(b []byte, seq int) {
	pType := b[0]
	pSubType := b[1]
	pLen := binary.LittleEndian.Uint16(b[2:])
	pData := b[4:pLen]
	fmt.Println(seq, "Type:", pType, "SubType:", pSubType, "Len:", pLen, "Data", hex.EncodeToString(pData))
	switch pType {
	case 1:
		fmt.Println("HWDev")
		parseHWDevice(b[:pLen])
	case 2:
		fmt.Println("ACPIDev")
		ParseACPI(b[:pLen])
	case 3:
		fmt.Println("MESSAGEDev")
		ParseMessaging(b[:pLen])
	case 4:
		fmt.Println("MEDIADev")
		ParseMedia(b[:pLen])
	case 5:
		fmt.Println("BIOSDev")
		ParseBIOS(b[:pLen])
	case 127:
		fmt.Println("EOF")
	default:
		fmt.Println("unimplemented Type", b[0])
	}

	next := b[pLen:]
	if len(next) > 0 {
		ParsePath(next, seq+1)
	}
}

func parseHWDevice(b []byte) {
	switch b[1] {
	case 1:
		fmt.Println("PCI")
		fmt.Println("Func", b[4])
		fmt.Println("Dev", b[5])
	case 2:
		fmt.Println("PCCARD")
	case 3:
		fmt.Println("Memory Mapped")
	case 4:
		fmt.Println("Vendor")
	default:
		fmt.Println("unimplemented hwdevice subtype", b[1])
	}
}

func ParseBIOS(b []byte) {
	switch b[1] {
	default:
	case 1:
		fmt.Println("BIOS Boot Specification Version 1.01")
		len := binary.LittleEndian.Uint16(b[2:])
		devType := binary.LittleEndian.Uint16(b[4:])
		statusFlag := binary.LittleEndian.Uint16(b[6:])
		desc := hex.EncodeToString(b[8:])
		fmt.Println(len, devType, statusFlag, desc)
	}
}

func ParseACPI(b []byte) {
	switch b[1] {
	case 1:
		fmt.Println("ACPI Device Path")
		len := binary.LittleEndian.Uint16(b[2:])
		hid := hex.EncodeToString(b[4:8])
		uid := hex.EncodeToString(b[8:12])
		rest := hex.EncodeToString(b[12:])
		fmt.Println("len:", len, "hid:", hid, "uid", uid, "rest", rest)
	case 2:
		fmt.Println("Expanded ACPI Device Path")
	case 3:
		fmt.Println("_ADR Device Path")
	default:
		fmt.Println("unimplemented acpi subtype", b[0])
	}
}

func ParseMessaging(b []byte) {
	switch b[1] {
	case 1:
		fmt.Println("ATAPI")
	case 2:
		fmt.Println("SCSI")
	case 3:
		fmt.Println("FIBRECHANNEL")
	case 4:
		fmt.Println("1394")
	case 5:
		fmt.Println("USB")
	case 6:
		fmt.Println("I2O Random Block Storage Class")
	case 9:
		fmt.Println("InfiniBand")
	case 10:
		len := binary.LittleEndian.Uint16(b[2:])
		guid, _ := uuid.FromBytes(EncodeUUID(b[4:20]))
		data, _ := uuid.FromBytes(EncodeUUID(b[20:36]))
		rest := hex.EncodeToString(b[36:])
		fmt.Println("Vendor", len, guid, data, rest)
	case 11:
		fmt.Println("MAC Address for a network interface")
	case 12:
		fmt.Println("IPv4")
	case 13:
		fmt.Println("IPv6")
	case 14:
		fmt.Println("UART")
	case 15:
		fmt.Println("USB Class")
	case 16:
		fmt.Println("USB WWID")
	case 17:
		fmt.Println("Device Logical unit")
	case 18:
		fmt.Println("SATA")
	case 19:
		fmt.Println("iSCSI")
	case 20:
		fmt.Println("Vlan")
	case 21:
		fmt.Println("FIBRECHANNEL_EX")
	case 22:
		fmt.Println("SAS_EX")
	case 23:
		fmt.Println("NVM Express Namespace")
	case 24:
		fmt.Println("Universal Resource Identifier (URI) Device Path")
	case 25:
		fmt.Println("UFS")
	case 26:
		fmt.Println("SD")
	case 27:
		fmt.Println("Bluetooth")
	case 28:
		fmt.Println("Wi-Fi Device Path")
	case 29:
		fmt.Println("eMMC")
	case 30:
		fmt.Println("BluetoothLE")
	default:
		fmt.Println("unimplemented subtype", b[0])
	}
}

func ParseMedia(b []byte) {
	switch b[1] {
	case 1:
		fmt.Println("Hard Drive")
		fmt.Println("PartitionNumber:", binary.LittleEndian.Uint32(b[4:]))
		fmt.Println("PartitionStart:", binary.LittleEndian.Uint64(b[8:]))
		fmt.Println("PartitionSize:", binary.LittleEndian.Uint64(b[16:]))
		fmt.Println("PartitionSignature:", b[24:40])
		fmt.Println("PartitionFormat:", b[40])
		fmt.Println("SignatureType:", b[41])
	case 2:
		fmt.Println("CD-ROM “El Torito”")
	case 4:
		fmt.Println("FilePath")

		fmt.Println(Utf16(b[4:]))
	case 6:
		fmt.Println("PIWG Firmware File")
	default:
		fmt.Println("unimplemented subtype", b[0])
	}
}
func Utf16(b []byte) string {
	str := make([]uint16, len(b)/2)
	for i := 0; i < len(b)-2; i += 2 {
		v := binary.LittleEndian.Uint16(b[i:])
		str[i/2] = v
	}
	return string(utf16.Decode(str))
}
func EncodeUUID(data []byte) []byte {
	ndata := make([]byte, 16)
	copy(ndata, data)
	binary.LittleEndian.PutUint32(ndata[0:], binary.BigEndian.Uint32(data[0:]))
	binary.LittleEndian.PutUint16(ndata[4:], binary.BigEndian.Uint16(data[4:]))
	binary.LittleEndian.PutUint16(ndata[6:], binary.BigEndian.Uint16(data[6:]))
	return ndata
}
