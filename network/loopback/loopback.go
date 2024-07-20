package loopback

import (
	"fmt"
	"net"
	"runtime"
)

type Loopback interface {
	Add() error
	Remove() error
}

func New(ip string, iface string) (Loopback, error) {
	nip := net.ParseIP(ip)
	if nip == nil {
		return nil, fmt.Errorf("Invalid IP %s", ip)
	}
	switch runtime.GOOS {
	case "darwin":
		return &LoopbackDarwin{nip, iface}, nil
	case "linux":
		return &LoopbackLinux{nip, iface}, nil
	}
	return nil, fmt.Errorf("Runtime is %s - can't setup loopback", runtime.GOOS)
}
