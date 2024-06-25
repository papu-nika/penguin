package penguin

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ping "github.com/prometheus-community/pro-bing"
)

const (
	RECIEVE       = 0
	SEND          = 1
	RECIEVE_ERROR = 2
)

type pingMag struct {
	msgType int
	pkt     *ping.Packet
	msg     string
}

type pinger struct {
	addr          string
	pinger        *ping.Pinger
	latestSuccess int
	history       []ping.Packet
}

type model struct {
	pinger []pinger
	sub    chan pingMag
	table  *table.Model
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func InitialModel(hosts []string, interval time.Duration) (*model, error) {
	var m model

	sub := make(chan pingMag)
	m.sub = sub

	var pingers []pinger
	for _, host := range hosts {
		p, err := ping.NewPinger(host)
		if err != nil {
			return nil, err
		}
		p.Interval = interval
		p.OnSend = func(p *ping.Packet) {
			sub <- pingMag{msgType: SEND}
		}
		p.OnRecv = func(pkt *ping.Packet) {
			sub <- pingMag{
				msgType: RECIEVE,
				pkt:     pkt,
				msg: fmt.Sprintf("%d bytes from %s:\ticmp_seq=%d time=%v",
					pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt,
				),
			}
		}
		p.OnRecvError = func(err error) {
			if strings.Contains(err.Error(), "read udp") {
				return
			}
			sub <- pingMag{
				msgType: RECIEVE_ERROR,
				msg:     fmt.Sprintf("Error: %s", err),
			}
		}
		p.OnSendError = func(pkt *ping.Packet, err error) {
			sub <- pingMag{
				msgType: RECIEVE_ERROR,
				msg:     fmt.Sprintf("Error: %s %d %s", err, pkt.Seq, pkt.Addr),
			}
		}
		pingers = append(pingers, pinger{pinger: p, latestSuccess: 0, addr: host})
	}
	m.pinger = pingers

	m.table = NewTable(len(hosts))
	return &m, nil
}

// A command that waits for the activity on the channel.
func waitForActivity(sub chan pingMag) tea.Cmd {
	return func() tea.Msg {
		return pingMag(<-sub)
	}
}

func (p *pinger) appendHistory(packet *ping.Packet) {
	p.history = append(p.history, *packet)
}

func findPinger(pingers []pinger, addr string) *pinger {
	for i, p := range pingers {
		if p.addr == addr {
			return &pingers[i]
		}
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return waitForActivity(m.sub)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		default:
			table, cmd := m.table.Update(msg)
			m.table = &table
			return m, cmd
		}

	case pingMag:
		switch msg.msgType {
		case SEND:
			return m, waitForActivity(m.sub) // wait for next event
		case RECIEVE:
			findPinger(m.pinger, msg.pkt.Addr).appendHistory(msg.pkt)
			return m, tea.Batch(
				waitForActivity(m.sub),
				tea.Println(msg.msg),
			)
		case RECIEVE_ERROR:
			return m, tea.Batch(
				waitForActivity(m.sub),
				tea.Println(msg.msg),
			)
		}
	}

	return m, nil
}

func (m model) View() string {
	var rows []table.Row
	for _, p := range m.pinger {
		rows = append(rows, createRow(p))
	}
	m.table.SetRows(rows)
	return baseStyle.Render(m.table.View())
}

func (m model) Run() error {
	for _, p := range m.pinger {
		go p.pinger.Run()
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return err
	}
	return nil
}
