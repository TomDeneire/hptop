package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Marker describes a single toner/ink cartridge level.
type Marker struct {
	Name  string
	Level int // percentage 0-100, or -1 if unknown
	Low   int // low-level threshold, or -1 if unknown
}

// Job describes a single queued CUPS print job.
type Job struct {
	ID   string
	Size string
	Date string
}

// Status is a full snapshot of the printer's reachability and health.
type Status struct {
	CheckedAt time.Time

	USBPresent bool
	USBBus     string
	USBDevice  string

	ServiceActive bool

	MDNSFound bool
	MDNSPort  int
	MDNSHost  string

	IPPReachable bool
	PrinterState string
	StateReasons []string
	Markers      []Marker

	Jobs []Job

	Notes []string
}

// RefreshStatus gathers a fresh snapshot of the printer's status.
func RefreshStatus(cfg Config) Status {
	st := Status{CheckedAt: time.Now()}

	st.USBPresent, st.USBBus, st.USBDevice = checkUSB(cfg.USBVendor, cfg.USBProduct)
	st.ServiceActive = checkServiceActive("ipp-usb")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.MDNSTimeout)
	defer cancel()
	st.MDNSFound, st.MDNSPort, st.MDNSHost = discoverMDNS(ctx, cfg.MDNSMatch)

	if st.MDNSFound {
		reachable, state, reasons, markers, err := queryIPP(st.MDNSPort, 5*time.Second)
		st.IPPReachable = reachable
		st.PrinterState = state
		st.StateReasons = reasons
		st.Markers = markers
		if err != nil {
			st.Notes = append(st.Notes, "IPP query failed: "+err.Error())
		}
	} else {
		st.PrinterState = "unreachable"
	}

	st.Jobs = checkQueue(cfg.CupsPrinter)

	if !st.USBPresent {
		st.Notes = append(st.Notes, "USB device not detected — check the cable and power.")
	} else if !st.ServiceActive {
		st.Notes = append(st.Notes, "The ipp-usb service is not active.")
	} else if !st.MDNSFound {
		st.Notes = append(st.Notes, "ipp-usb has not claimed the device — try unplugging and reconnecting the USB cable.")
	}

	for _, r := range st.StateReasons {
		switch {
		case strings.Contains(r, "error"):
			st.Notes = append(st.Notes, "Printer reports an error condition: "+r)
		case strings.Contains(r, "toner-low") || strings.Contains(r, "toner-empty"):
			st.Notes = append(st.Notes, "Toner cartridge warning: "+r)
		}
	}

	return st
}

// checkUSB reports whether a USB device with the given vendor/product
// (hex, no 0x prefix) is currently attached.
func checkUSB(vendor, product string) (present bool, bus, dev string) {
	vendor = strings.ToLower(strings.TrimSpace(vendor))
	product = strings.ToLower(strings.TrimSpace(product))

	matches, err := filepath.Glob("/sys/bus/usb/devices/*/idVendor")
	if err != nil {
		return false, "", ""
	}

	for _, vendorFile := range matches {
		dir := filepath.Dir(vendorFile)

		v, err := readSysfsValue(vendorFile)
		if err != nil || v != vendor {
			continue
		}

		p, err := readSysfsValue(filepath.Join(dir, "idProduct"))
		if err != nil || p != product {
			continue
		}

		busnum, _ := readSysfsValue(filepath.Join(dir, "busnum"))
		devnum, _ := readSysfsValue(filepath.Join(dir, "devnum"))
		return true, busnum, devnum
	}

	return false, "", ""
}

func readSysfsValue(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(string(data))), nil
}

// checkServiceActive reports whether a systemd unit is currently active.
func checkServiceActive(unit string) bool {
	out, err := exec.Command("systemctl", "is-active", unit).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// checkQueue returns the jobs currently queued for the given CUPS printer.
func checkQueue(printer string) []Job {
	out, err := exec.Command("lpstat", "-o", printer).Output()
	if err != nil {
		return nil
	}

	var jobs []Job
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		jobs = append(jobs, Job{
			ID:   fields[0],
			Size: fields[2],
			Date: strings.Join(fields[3:], " "),
		})
	}
	return jobs
}
