// Copyright (c) 2022 Arthur Skowronek <0x5a17ed@tuta.io> and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// <https://www.apache.org/licenses/LICENSE-2.0>
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package efidevicepath

import (
	"fmt"
	"io"
	"net"

	"github.com/blindson76/uefi/efi/efireader"
)

const (
	URIDeviceSubType    = 10
	MACAddressSubType   = 11
	IPv4DeviceSubType   = 12
	SATADeviceSubType   = 18
	VENDORDeviceSubType = 24
)

func ParseMessagingDevicePath(r io.Reader, h Head) (p DevicePath, err error) {
	switch h.SubType {
	case MACAddressSubType:
		p = &MACAddressDevicePath{Head: h}
	case IPv4DeviceSubType:
		p = &IPv4DevicePath{Head: h}
	case SATADeviceSubType:
		p = &SATADevicePath{Head: h}
	case URIDeviceSubType:
		p = &URIDevicePath{Head: h}
	default:
		p = &UnrecognizedDevicePath{Head: h}
	}
	_, err = p.ReadFrom(r)
	return
}

type MACAddressDevicePath struct {
	Head
	MAC      [32]byte
	AddrType byte
}

func (p *MACAddressDevicePath) Text() string {
	return fmt.Sprintf("MAC Address: %s Type: %d", net.HardwareAddr(p.MAC[:6]).String(), p.AddrType)
}

func (p *MACAddressDevicePath) GetHead() *Head {
	return &p.Head
}

func (p *MACAddressDevicePath) ReadFrom(r io.Reader) (n int64, err error) {
	fr := efireader.NewFieldReader(r, &n)

	if err = fr.ReadFields(&p.MAC, &p.AddrType); err != nil {
		return
	}
	return
}

type SATADevicePath struct {
	Head
	HBAPortNumber     uint16
	PortMulPortNumber uint16
	LUN               uint16
}

func (p *SATADevicePath) Text() string {
	return fmt.Sprintf("SATA Port: %d PortMul: %d LUN: %d", p.HBAPortNumber, p.PortMulPortNumber, p.LUN)
}

func (p *SATADevicePath) GetHead() *Head {
	return &p.Head
}

func (p *SATADevicePath) ReadFrom(r io.Reader) (n int64, err error) {
	fr := efireader.NewFieldReader(r, &n)

	if err = fr.ReadFields(&p.HBAPortNumber, &p.PortMulPortNumber, &p.LUN); err != nil {
		return
	}
	return
}

type IPv4DevicePath struct {
	Head
	LocalIP     net.IP
	RemoteIP    net.IP
	LocalPort   uint16
	RemotePort  uint16
	Protocol    uint16
	Static      bool
	GatewayAddr net.IP
	SubnetAddr  net.IP
}

func (p *IPv4DevicePath) Text() string {
	return fmt.Sprintf("IPv4 Local:%s Remote:%s", p.LocalIP.String(), p.RemoteIP.String())
}

func (p *IPv4DevicePath) GetHead() *Head {
	return &p.Head
}

func (p *IPv4DevicePath) ReadFrom(r io.Reader) (n int64, err error) {
	fr := efireader.NewFieldReader(r, &n)
	p.LocalIP = make(net.IP, 4)
	p.RemoteIP = make(net.IP, 4)
	p.SubnetAddr = make(net.IP, 4)
	p.GatewayAddr = make(net.IP, 4)

	if err = fr.ReadFields(&p.LocalIP, &p.RemoteIP, &p.LocalPort, &p.RemotePort); err != nil {
		return
	}
	return
}

type URIDevicePath struct {
	Head
	URI []byte
}

func (p *URIDevicePath) Text() string {
	return fmt.Sprintf("URI: %s", string(p.URI))
}

func (p *URIDevicePath) GetHead() *Head {
	return &p.Head
}

func (p *URIDevicePath) ReadFrom(r io.Reader) (n int64, err error) {
	fr := efireader.NewFieldReader(r, &n)
	p.URI = make([]byte, p.Length-4)

	if err = fr.ReadFields(&p.URI); err != nil {
		return
	}
	return
}
