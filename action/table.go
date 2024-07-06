package penguin

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

const (
	OK_SYNBOL   = "✅"
	NG_SYNBOL   = "❌"
	NONE_SYNBOL = "🔲"
)

func stringOKNONE(isRecieved bool) string {
	if isRecieved {
		return OK_SYNBOL
	}
	return NONE_SYNBOL
}

func NewTable(hight int) *table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(hight),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	return &t
}

var columns = []table.Column{
	{Title: "", Width: 2},
	{Title: "Host", Width: 22},
	{Title: "Packets", Width: 8},
	{Title: "MinRtt", Width: 12},
	{Title: "MaxRtt", Width: 12},
	{Title: "AvgRtt", Width: 12},
	{Title: "StdDev", Width: 12},
	{Title: "History", Width: 42},
}

func createRow(p pinger) table.Row {
	stat := p.pinger.Statistics()

	var history []string
	for _, h := range p.history.Data {
		if h.isRecieved {
			history = append(history, OK_SYNBOL)
		} else {
			history = append(history, NONE_SYNBOL)
		}
	}

	latest := stringOKNONE(p.history.IsSuccessIndex(p.history.Len() - 1))

	var host string
	if stat.Addr == stat.IPAddr.IP.String() {
		host = stat.Addr
	} else {
		host = fmt.Sprintf("%s(%s)", stat.Addr, stat.IPAddr.IP.String())
	}

	return table.Row{
		latest,
		host,
		fmt.Sprintf("%d/%d", stat.PacketsRecv, stat.PacketsSent),
		stat.MinRtt.String(),
		stat.AvgRtt.String(),
		stat.MaxRtt.String(),
		stat.StdDevRtt.String(),
		strings.Join(history, ""),
	}
}
