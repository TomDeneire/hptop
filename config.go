package main

import "time"

// Config holds the runtime configuration for monitoring one printer.
type Config struct {
	CupsPrinter     string
	USBVendor       string
	USBProduct      string
	MDNSMatch       string
	MDNSTimeout     time.Duration
	RefreshInterval time.Duration
}
