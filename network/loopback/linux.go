package loopback

import (
	"fmt"
	"log"
	"net"
	"os/exec"
)

type LoopbackLinux struct {
	Alias     net.IP
	Interface string
}

func (l *LoopbackLinux) Add() error {
	ip := l.Alias.String()
	log.Printf("Setting up loopback IP %s", ip)
	c := exec.Command("ip", "addr", "replace", ip, "dev", l.Interface)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s %s", err, out, err)
	}
	if len(out) != 0 {
		return fmt.Errorf("ifconfig error: %s", out)
	}
	return nil
}

func (l *LoopbackLinux) Remove() error {
	ip := l.Alias.String()
	log.Printf("Removing loopback IP %s", ip)
	c := exec.Command("ip", "addr", "delete", ip, "dev", l.Interface)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s %s", err, out, err)
	}
	if len(out) != 0 {
		return fmt.Errorf("ifconfig error: %s", out)
	}
	return nil
}
