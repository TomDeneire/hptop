package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Margin(0, 1, 1, 0)
	headerLabel = lipgloss.NewStyle().Bold(true)
)

type statusMsg Status

type tickMsg struct{}

type model struct {
	cfg     Config
	status  Status
	loading bool
	width   int
}

func initialModel(cfg Config) model {
	return model{cfg: cfg, loading: true}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchStatusCmd(m.cfg), tickCmd(m.cfg.RefreshInterval))
}

func fetchStatusCmd(cfg Config) tea.Cmd {
	return func() tea.Msg {
		return statusMsg(RefreshStatus(cfg))
	}
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "c", "r":
			m.loading = true
			return m, fetchStatusCmd(m.cfg)
		}

	case tickMsg:
		m.loading = true
		return m, tea.Batch(fetchStatusCmd(m.cfg), tickCmd(m.cfg.RefreshInterval))

	case statusMsg:
		m.status = Status(msg)
		m.loading = false
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("hptop") + dimStyle.Render("  —  "+m.cfg.CupsPrinter) + "\n")

	checked := "never"
	if !m.status.CheckedAt.IsZero() {
		checked = m.status.CheckedAt.Format("15:04:05")
	}
	loadingTag := ""
	if m.loading {
		loadingTag = warnStyle.Render("  checking…")
	}
	b.WriteString(dimStyle.Render("last check: "+checked) + loadingTag + "\n\n")

	b.WriteString(panelStyle.Render(m.connectionPanel()) + "\n")
	b.WriteString(panelStyle.Render(m.statePanel()) + "\n")
	b.WriteString(panelStyle.Render(m.markersPanel()) + "\n")
	b.WriteString(panelStyle.Render(m.queuePanel()) + "\n")

	if len(m.status.Notes) > 0 {
		b.WriteString(panelStyle.Render(m.notesPanel()) + "\n")
	}

	b.WriteString(dimStyle.Render("[c] check now   [q] quit"))

	return b.String()
}

func (m model) connectionPanel() string {
	var b strings.Builder
	b.WriteString(headerLabel.Render("Connection") + "\n")
	b.WriteString(statusLine("USB device", m.status.USBPresent, func() string {
		if m.status.USBPresent {
			return fmt.Sprintf("bus %s dev %s", m.status.USBBus, m.status.USBDevice)
		}
		return "not detected"
	}()))
	b.WriteString(statusLine("ipp-usb service", m.status.ServiceActive, boolLabel(m.status.ServiceActive, "active", "inactive")))
	b.WriteString(statusLine("mDNS advertised", m.status.MDNSFound, func() string {
		if m.status.MDNSFound {
			return fmt.Sprintf("port %d", m.status.MDNSPort)
		}
		return "not found"
	}()))
	b.WriteString(statusLine("IPP reachable", m.status.IPPReachable, boolLabel(m.status.IPPReachable, "yes", "no")))
	return b.String()
}

func (m model) statePanel() string {
	var b strings.Builder
	b.WriteString(headerLabel.Render("Printer state") + "\n")

	style := dimStyle
	switch m.status.PrinterState {
	case "idle":
		style = okStyle
	case "processing":
		style = warnStyle
	case "stopped", "unreachable":
		style = errStyle
	}
	state := m.status.PrinterState
	if state == "" {
		state = "unknown"
	}
	b.WriteString(style.Render(state) + "\n")

	if len(m.status.StateReasons) == 0 {
		b.WriteString(dimStyle.Render("(no reasons reported)"))
	} else {
		for _, r := range m.status.StateReasons {
			b.WriteString(warnStyle.Render("• " + r + "\n"))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) markersPanel() string {
	var b strings.Builder
	b.WriteString(headerLabel.Render("Toner levels") + "\n")

	if len(m.status.Markers) == 0 {
		b.WriteString(dimStyle.Render("no data"))
		return b.String()
	}

	for _, mk := range m.status.Markers {
		b.WriteString(fmt.Sprintf("%-18s %s\n", capitalize(mk.Name), renderBar(mk.Level, mk.Low)))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) queuePanel() string {
	var b strings.Builder
	b.WriteString(headerLabel.Render("Print queue") + "\n")

	if len(m.status.Jobs) == 0 {
		b.WriteString(okStyle.Render("no active jobs"))
		return b.String()
	}

	for _, j := range m.status.Jobs {
		b.WriteString(fmt.Sprintf("#%s  %s  %s\n", j.ID, j.Size, j.Date))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) notesPanel() string {
	var b strings.Builder
	b.WriteString(headerLabel.Render("Notes") + "\n")
	for _, n := range m.status.Notes {
		b.WriteString(warnStyle.Render("• "+n) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func statusLine(label string, ok bool, detail string) string {
	mark := okStyle.Render("✓")
	if !ok {
		mark = errStyle.Render("✗")
	}
	return fmt.Sprintf("%s %-18s %s\n", mark, label, dimStyle.Render(detail))
}

func boolLabel(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

func renderBar(pct, low int) string {
	const width = 20
	if pct < 0 {
		return dimStyle.Render(strings.Repeat("░", width) + " unknown")
	}

	filled := pct * width / 100
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	style := okStyle
	threshold := low
	if threshold < 0 {
		threshold = 15
	}
	switch {
	case pct <= threshold:
		style = errStyle
	case pct <= threshold*2:
		style = warnStyle
	}

	return style.Render(bar) + fmt.Sprintf(" %3d%%", pct)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
