# hptop

A small terminal status monitor for an HP printer connected via USB through
`ipp-usb`, in the spirit of `btop`. Built with [Charm](https://charm.sh)'s
`bubbletea` and `lipgloss`.

It reports, at a glance:

- whether the USB device is detected
- whether the `ipp-usb` systemd service is active
- whether the printer is advertised over mDNS (and on which port)
- whether it answers IPP requests, and its reported state
- toner/ink levels per cartridge
- the current CUPS print queue
- plain-language notes when something looks wrong (USB unplugged,
  `ipp-usb` hasn't claimed the device, low toner, error states, ...)

## Build

```sh
make build
```

## Run

`make build` installs the binary to `$HOME/bin`, so run it from there (make
sure `$HOME/bin` is on your `PATH`):

```sh
hptop
```

Keys:

- `c` or `r` — check now
- `q` / `ctrl+c` / `esc` — quit

The status also refreshes automatically every 30 seconds.

## Flags

| Flag           | Default                      | Meaning                                                 |
| -------------- | ---------------------------- | ------------------------------------------------------- |
| `-printer`     | `HP-Color-LaserJet-Pro-3202` | CUPS printer queue name                                 |
| `-usb-vendor`  | `03f0`                       | USB vendor ID (hex, no `0x` prefix)                     |
| `-usb-product` | `1b00`                       | USB product ID (hex, no `0x` prefix)                    |
| `-mdns-match`  | `3202`                       | Substring to match in the `_ipp._tcp` mDNS service name |
| `-interval`    | `30`                         | Auto-refresh interval in seconds                        |

## Notes

- `ipp-usb` advertises its local IPP bridge on the loopback interface by
  default, which most mDNS client libraries skip when browsing — `hptop`
  explicitly includes it.
- Requires `systemctl`, `lpstat` and read access to `/sys/bus/usb/devices`;
  no `sudo` is needed for any of the checks it performs.
