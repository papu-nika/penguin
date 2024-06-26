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
	RECEIVE = iota
	SEND
	RECEIVE_ERROR
	STOP
	RESET
)

type pingMsg struct {
	msgType int
	pkt     *ping.Packet
	msg     string
}

type pinger struct {
	// addr          string
	pinger        *ping.Pinger
	latestSuccess int
	history       History
}

type model struct {
	pinger map[string]*pinger
	sub    chan pingMsg
	table  *table.Model
	hosts  []string
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func InitialModel(hosts []string, interval time.Duration) (*model, error) {
	var m model

	sub := make(chan pingMsg)
	m.sub = sub

	var pingers map[string]*pinger = make(map[string]*pinger, len(hosts))
	for _, host := range hosts {
		p, err := ping.NewPinger(host)
		if err != nil {
			return nil, err
		}
		p.Interval = interval
		p.OnSend = func(p *ping.Packet) {
			sub <- pingMsg{msgType: SEND, pkt: p}
		}
		p.OnRecv = func(pkt *ping.Packet) {
			sub <- pingMsg{
				msgType: RECEIVE,
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
			sub <- pingMsg{
				msgType: RECEIVE_ERROR,
				msg:     errorPacket(err, nil),
			}
		}
		p.OnSendError = func(pkt *ping.Packet, err error) {
			sub <- pingMsg{
				msgType: RECEIVE_ERROR,
				msg:     errorPacket(err, pkt),
			}
		}

		pingers[host] = &pinger{
			pinger:        p,
			latestSuccess: 0,
			history:       History{MaxLen: 10},
		}
	}
	m.pinger = pingers
	m.hosts = hosts
	m.table = NewTable(len(hosts))

	return &m, nil
}

// A command that waits for the activity on the channel.
func waitForActivity(sub chan pingMsg) tea.Cmd {
	return func() tea.Msg {
		return pingMsg(<-sub)
	}
}

func (m model) findPinger(addr string) *pinger {
	p := m.pinger[addr]
	return p
}

func (m model) Init() tea.Cmd {
	return waitForActivity(m.sub)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			for _, p := range m.pinger {
				p.pinger.Stop()
			}
			return m, tea.Quit
		default:
			table, cmd := m.table.Update(msg)
			m.table = &table
			return m, cmd
		}

	case pingMsg:
		switch msg.msgType {
		case SEND:
			m.findPinger(msg.pkt.Addr).history.AppendFirst(HistoryData{Seq: msg.pkt.Seq, isRecieved: false})
			return m, waitForActivity(m.sub) // wait for next event
		case RECEIVE:
			m.findPinger(msg.pkt.Addr).history.Recieved(msg.pkt.Seq)
			return m, tea.Batch(
				waitForActivity(m.sub),
				tea.Println(msg.msg),
			)
		case RECEIVE_ERROR:
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
	for _, host := range m.hosts {
		rows = append(rows, createRow(*m.pinger[host]))
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

func errorPacket(err error, pkt *ping.Packet) string {
	if pkt == nil {
		return fmt.Sprintf("Error: %s", err)
	}
	return fmt.Sprintf("Error: %s %d %s", err, pkt.Seq, pkt.Addr)
}
