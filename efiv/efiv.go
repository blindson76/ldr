package efiv

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
)

type IFilePath interface {
}

func NewFilePath(b []byte) IFilePath {
	fp := &FilePath{
		Type:    b[0],
		SubType: b[1],
		Len:     binary.LittleEndian.Uint16(b[2:]),
	}
	fp.Data = b[4:fp.Len]
	switch fp.Type {
	case 4:
		return ParseMediaDevice(fp, b)
	}
	log.Println("not implemented type", fp.Type)
	return fp
}

type FilePath struct {
	Type    byte
	SubType byte
	Len     uint16
	Data    []byte
}

func (fp *FilePath) String() string {
	return fmt.Sprintf("Type %d, SubType %d, Len %d, Data %s", fp.Type, fp.SubType, fp.Len, hex.EncodeToString(fp.Data))
}

type HardDriveMediaDevicePath struct {
	IFilePath
	FilePath
	PartitionNumber    uint32
	PartitionStart     uint64
	PartitionSize      uint64
	PartitionSignature []byte
	PartitionFormat    byte
	SignatureType      byte
}

func NewHardDriveMediaDevicePath(header *FilePath, b []byte) *HardDriveMediaDevicePath {
	return &HardDriveMediaDevicePath{
		FilePath:           *header,
		PartitionNumber:    binary.LittleEndian.Uint32(b[4:]),
		PartitionStart:     binary.LittleEndian.Uint64(b[8:]),
		PartitionSize:      binary.LittleEndian.Uint64(b[16:]),
		PartitionSignature: b[24:40],
		PartitionFormat:    b[40],
		SignatureType:      b[41],
	}
}

func ParseMediaDevice(header *FilePath, b []byte) IFilePath {
	switch header.SubType {
	case 1:
		return NewHardDriveMediaDevicePath(header, b)
	}

	log.Println("Unimplemented media subtype ", header.SubType)
	return header
}
