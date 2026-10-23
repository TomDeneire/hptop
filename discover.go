package main

import (
	"context"
	"net"
	"strings"

	"github.com/grandcat/zeroconf"
)

// discoverMDNS browses for an _ipp._tcp mDNS service whose instance name
// contains match (case-insensitive), returning its port and hostname.
//
// ipp-usb advertises its local IPP bridge on the loopback interface by
// default (see /etc/ipp-usb/ipp-usb.conf), which zeroconf's default
// interface enumeration skips. So we explicitly include it alongside any
// other multicast-capable interfaces.
func discoverMDNS(ctx context.Context, match string) (found bool, port int, host string) {
	resolver, err := zeroconf.NewResolver(zeroconf.SelectIfaces(multicastInterfaces()))
	if err != nil {
		return false, 0, ""
	}

	entries := make(chan *zeroconf.ServiceEntry, 16)
	result := make(chan *zeroconf.ServiceEntry, 1)
	match = strings.ToLower(match)

	go func() {
		for entry := range entries {
			if strings.Contains(strings.ToLower(entry.Instance), match) {
				select {
				case result <- entry:
				default:
				}
			}
		}
	}()

	if err := resolver.Browse(ctx, "_ipp._tcp", "local.", entries); err != nil {
		return false, 0, ""
	}

	select {
	case entry := <-result:
		return true, entry.Port, entry.HostName
	case <-ctx.Done():
		return false, 0, ""
	}
}

// multicastInterfaces returns all up interfaces that can carry mDNS
// traffic, including loopback (where ipp-usb advertises by default).
func multicastInterfaces() []net.Interface {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var out []net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast != 0 || iface.Flags&net.FlagLoopback != 0 {
			out = append(out, iface)
		}
	}
	return out
}
