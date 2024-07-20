package loopback

import (
	"fmt"
	"log"
	"net"
	"os/exec"
)

type LoopbackDarwin struct {
	Alias     net.IP
	Interface string
}

func (l *LoopbackDarwin) Add() error {
	ip := l.Alias.String()
	log.Printf("Setting up loopback IP %s", ip)
	c := exec.Command("ifconfig", l.Interface, "alias", ip)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s %s", err, out, err)
	}
	if len(out) != 0 {
		return fmt.Errorf("ifconfig error: %s", out)
	}
	return nil
}

func (l *LoopbackDarwin) Remove() error {
	ip := l.Alias.String()
	log.Printf("Removing loopback IP %s", ip)
	c := exec.Command("ifconfig", l.Interface, "delete", ip)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s %s", err, out, err)
	}
	if len(out) != 0 {
		return fmt.Errorf("ifconfig error: %s", out)
	}
	return nil
}
