package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/OpenPrinting/goipp"
)

// printer-state enum values, per RFC 8011 4.4.11.
const (
	stateIdle       = 3
	stateProcessing = 4
	stateStopped    = 5
)

// queryIPP asks the printer (reachable via ipp-usb on localhost:port) for its
// current state and marker (toner/ink) levels.
func queryIPP(port int, timeout time.Duration) (reachable bool, state string, reasons []string, markers []Marker, err error) {
	uri := fmt.Sprintf("http://localhost:%d/ipp/print", port)

	req := goipp.NewRequest(goipp.DefaultVersion, goipp.OpGetPrinterAttributes, 1)
	req.Operation.Add(goipp.MakeAttribute("attributes-charset", goipp.TagCharset, goipp.String("utf-8")))
	req.Operation.Add(goipp.MakeAttribute("attributes-natural-language", goipp.TagLanguage, goipp.String("en")))
	req.Operation.Add(goipp.MakeAttribute("printer-uri", goipp.TagURI, goipp.String(uri)))

	wanted := goipp.MakeAttribute("requested-attributes", goipp.TagKeyword, goipp.String("printer-state"))
	wanted.Values.Add(goipp.TagKeyword, goipp.String("printer-state-reasons"))
	wanted.Values.Add(goipp.TagKeyword, goipp.String("marker-names"))
	wanted.Values.Add(goipp.TagKeyword, goipp.String("marker-levels"))
	wanted.Values.Add(goipp.TagKeyword, goipp.String("marker-low-levels"))
	req.Operation.Add(wanted)

	data, encErr := req.EncodeBytes()
	if encErr != nil {
		return false, "", nil, nil, encErr
	}

	client := &http.Client{Timeout: timeout}
	resp, httpErr := client.Post(uri, goipp.ContentType, bytes.NewReader(data))
	if httpErr != nil {
		return false, "", nil, nil, httpErr
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return false, "", nil, nil, readErr
	}

	var respMsg goipp.Message
	if decErr := respMsg.DecodeBytes(body); decErr != nil {
		return false, "", nil, nil, decErr
	}

	names := ippStrings(respMsg, "marker-names")
	levels := ippInts(respMsg, "marker-levels")
	lows := ippInts(respMsg, "marker-low-levels")

	for i, name := range names {
		level, low := -1, -1
		if i < len(levels) {
			level = levels[i]
		}
		if i < len(lows) {
			low = lows[i]
		}
		markers = append(markers, Marker{Name: name, Level: level, Low: low})
	}

	return true, ippPrinterState(respMsg), nonNoneReasons(ippStrings(respMsg, "printer-state-reasons")), markers, nil
}

// nonNoneReasons strips the "none" placeholder IPP printers report when
// printer-state-reasons has no real conditions to list (RFC 8011 5.4.12).
func nonNoneReasons(reasons []string) []string {
	var out []string
	for _, r := range reasons {
		if r != "none" {
			out = append(out, r)
		}
	}
	return out
}

func ippPrinterState(msg goipp.Message) string {
	for _, a := range msg.Printer {
		if a.Name != "printer-state" || len(a.Values) == 0 {
			continue
		}
		if v, ok := a.Values[0].V.(goipp.Integer); ok {
			switch int(v) {
			case stateIdle:
				return "idle"
			case stateProcessing:
				return "processing"
			case stateStopped:
				return "stopped"
			default:
				return fmt.Sprintf("unknown (%d)", int(v))
			}
		}
	}
	return "unknown"
}

func ippStrings(msg goipp.Message, name string) []string {
	var out []string
	for _, a := range msg.Printer {
		if a.Name != name {
			continue
		}
		for _, v := range a.Values {
			out = append(out, v.V.String())
		}
	}
	return out
}

func ippInts(msg goipp.Message, name string) []int {
	var out []int
	for _, a := range msg.Printer {
		if a.Name != name {
			continue
		}
		for _, v := range a.Values {
			if iv, ok := v.V.(goipp.Integer); ok {
				out = append(out, int(iv))
			}
		}
	}
	return out
}
