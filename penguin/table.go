package penguin

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

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
	{Title: "Host", Width: 16},
	{Title: "Packets", Width: 8},
	{Title: "MinRtt", Width: 12},
	{Title: "MaxRtt", Width: 12},
	{Title: "AvgRtt", Width: 12},
	{Title: "StdDev", Width: 12},
	{Title: "History", Width: 24},
}

func createRow(p pinger) table.Row {
	stat := p.pinger.Statistics()

	var latest string
	if p.latestSuccess < stat.PacketsRecv {
		latest = "✅"
	} else {
		latest = "❌"
	}
	p.latestSuccess = stat.PacketsRecv

	var history []string
	for _, h := range p.history {
		history = append(history, fmt.Sprintf("%d", h.Seq))
	}

	return table.Row{
		latest,
		stat.Addr,
		fmt.Sprintf("%d/%d", stat.PacketsRecv, stat.PacketsSent),
		stat.MinRtt.String(),
		stat.AvgRtt.String(),
		stat.MaxRtt.String(),
		stat.StdDevRtt.String(),
		strings.Join(history, ""),
	}
}
