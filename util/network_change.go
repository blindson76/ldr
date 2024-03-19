package util

import (
	"net"
)

type NetworkListener interface {
	NetworkChanged(bool, net.Addr)
}
type NetworkChangeNotifier struct {
	observers map[NetworkListener]string
	status    map[NetworkListener]net.Addr
	emits     chan struct{}
	stop      chan struct{}
}

func NewNetworkChangeNotifier() *NetworkChangeNotifier {
	notifier := &NetworkChangeNotifier{
		observers: map[NetworkListener]string{},
		status:    map[NetworkListener]net.Addr{},
		emits:     make(chan struct{}),
		stop:      make(chan struct{}),
	}
	return notifier
}
func (n *NetworkChangeNotifier) networkChanged() {
	for k, address := range n.observers {

		iface, addr := InterfaceByAddress(address)
		if iface != nil && (iface.Flags&net.FlagUp) > 0 && (iface.Flags&net.FlagRunning) > 0 {
			if n.status[k] == nil || n.status[k].String() != addr.String() {
				n.status[k] = addr
				k.NetworkChanged(true, addr)
			}

		} else {
			if n.status[k] != nil {
				n.status[k] = nil
				k.NetworkChanged(false, nil)
			}
		}
	}

}
func (n *NetworkChangeNotifier) Subscribe(observer NetworkListener, address string) {
	n.observers[observer] = address
	n.status[observer] = nil

}
func (n *NetworkChangeNotifier) Start() {

	go func() {
		for {
			select {
			case <-n.emits:
				n.networkChanged()
			case <-n.stop:
				return
			}
		}
	}()

	n.init()
}
