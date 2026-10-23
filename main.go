// Command hptop is a terminal status monitor for an HP printer connected
// over USB via ipp-usb, inspired by tools like btop.
//
// It reports USB presence, the ipp-usb driverless bridge, toner levels,
// printer state and the CUPS print queue, refreshing automatically and
// on demand ('c' or 'r').
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var cfg Config
	var refreshSec int

	flag.StringVar(&cfg.CupsPrinter, "printer", "HP-Color-LaserJet-Pro-3202", "CUPS printer queue name")
	flag.StringVar(&cfg.USBVendor, "usb-vendor", "03f0", "USB vendor ID (hex, no 0x prefix)")
	flag.StringVar(&cfg.USBProduct, "usb-product", "1b00", "USB product ID (hex, no 0x prefix)")
	flag.StringVar(&cfg.MDNSMatch, "mdns-match", "3202", "substring to match in the mDNS _ipp._tcp service name")
	flag.IntVar(&refreshSec, "interval", 30, "auto-refresh interval in seconds")
	flag.Parse()

	cfg.RefreshInterval = time.Duration(refreshSec) * time.Second
	cfg.MDNSTimeout = 4 * time.Second

	p := tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "hptop: "+err.Error())
		os.Exit(1)
	}
}
