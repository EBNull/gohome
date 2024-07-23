package network

import (
	"errors"
	"flag"

	"github.com/ebnull/gohome/network/hostfile"
	"github.com/ebnull/gohome/network/loopback"
)

var flagLoopbackInterface = flag.String("loopback-interface", "lo0", "Specifies the loopback adapter interface for --auto mode")
var flagHostfile = flag.String("hostfile", "/etc/hosts", "Specifies the location of the hostfile to edit for --auto mode")

type HostAliasManager struct {
	lb loopback.Loopback
	h  *hostfile.Hostfile
	he *hostfile.HostEntry
}

func NewAliasManager(host string, ip string) (*HostAliasManager, error) {
	lb, err := loopback.New(ip, *flagLoopbackInterface)
	if err != nil {
		return nil, err
	}
	eh := &hostfile.Hostfile{*flagHostfile}
	he, err := hostfile.NewHostEntry(host, ip)
	if err != nil {
		return nil, err
	}
	return &HostAliasManager{lb, eh, he}, nil
}

func (am *HostAliasManager) Start() (error, func() error) {
	err := am.lb.Add()
	if err != nil {
		return err, func() error { return nil }
	}
	undo := func() error {
		return am.lb.Remove()
	}
	he, err := am.h.HostExists(am.he.IP.String())
	if err != nil {
		return err, undo
	}
	if !he {
		err = am.h.AddHost(am.he, "added by gohome")
		if err != nil {
			return err, undo
		}
	}
	return nil, func() error {
		_, err := am.h.RemoveHost(am.he)
		return errors.Join(undo(), err)
	}
}

func (am *HostAliasManager) Host() string {
	return am.he.Host
}
func (am *HostAliasManager) Exists() (bool, error) {
	return am.he.Exists()
}
