package hostfile

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/google/renameio/v2"
)

type Hostfile struct {
	Filename string
}

func (eh *Hostfile) HostExists(ip string) (bool, error) {
	// We should probably do this right by parsing the file, but this works (unless the IP is in a comment)
	d, err := os.ReadFile(eh.Filename)
	if err != nil {
		return false, err
	}
	if bytes.Contains(d, []byte(ip)) {
		return true, nil
	}
	return false, nil
}

func (eh *Hostfile) AddHost(e *HostEntry, comment string) error {
	ip := e.IP.String()
	log.Printf("Adding hosts entry %s %s", ip, e.Host)
	// We can't add an IP twice, so we check the lines ahead of time and bail if that would happen.
	he, err := eh.HostExists(ip)
	if err != nil {
		return err
	}
	if he {
		return fmt.Errorf("%s already contains the IP address %s", eh.Filename, ip)
	}
	err = editFileLines(eh.Filename, []string{fmt.Sprintf("\n%s %s # %s", e.IP.String(), e.Host, comment)}, func(s string) bool { return false })
	if err != nil {
		return err
	}
	return nil
}

func editFileLines(filename string, appendLines []string, removeLineFunc func(line string) bool) error {
	// We'd like to keep comments, so we edit line by line

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	linebuf := []string{}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if !removeLineFunc(line) {
			linebuf = append(linebuf, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	linebuf = append(linebuf, appendLines...)

	file.Close()

	sbuf := strings.Join(linebuf, "\n")

	// Use an atomic write - a torn /etc/hosts is bad(tm)
	return renameio.WriteFile(filename, []byte(sbuf), 0644)
}

func (eh *Hostfile) RemoveHost(e *HostEntry) (bool, error) {
	log.Printf("Removing hosts entry %s %s", e.IP, e.Host)
	removed := false
	return removed, editFileLines(eh.Filename, []string{}, func(line string) bool {
		// Be paranoid and make sure we're only removing an ip and host that we expect, and that there's no other hosts here
		left, _, _ := strings.Cut(line, "#")
		dataline := strings.TrimSpace(left)
		f := strings.Fields(dataline)
		if len(f) == 2 { // Line has only one hostname
			if f[0] == e.IP.String() { // And our IP
				if f[1] == e.Host { // That is ours
					removed = true
					return true
				}
			}
		}
		return false
	})
}

type HostEntry struct {
	Host string
	IP   net.IP
}

func NewHostEntry(host string, ip string) (*HostEntry, error) {
	nip := net.ParseIP(ip)
	if nip == nil {
		return nil, fmt.Errorf("Invalid IP address %s", nip)
	}
	return &HostEntry{host, nip}, nil
}

func (e *HostEntry) Exists() (bool, error) {
	ips, err := net.LookupIP(e.Host)
	if err != nil {
		return false, err
	}
	for _, i := range ips {
		if i.Equal(e.IP) {
			return true, nil
		}
	}
	return false, nil
}
