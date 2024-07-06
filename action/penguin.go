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
	pinger  *ping.Pinger
	history History
}

type model struct {
	pinger map[string]*pinger
	// 順序を保持するためのスライス
	pingerList []*pinger
	sub        chan pingMsg
	table      *table.Model
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
					pkt.Nbytes, pkt.Addr, pkt.Seq, pkt.Rtt,
				),
			}
		}
		p.OnRecvError = func(err error) {
			if strings.Contains(err.Error(), "read udp") || strings.Contains(err.Error(), "read ip4") {
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

		SetPrivileged(p)
		pingers[p.Statistics().IPAddr.IP.String()] = &pinger{
			pinger:  p,
			history: History{MaxLen: 20},
		}
		m.pingerList = append(m.pingerList, pingers[p.Statistics().IPAddr.IP.String()])

	}
	m.pinger = pingers
	m.table = NewTable(len(hosts))

	return &m, nil
}

// A command that waits for the activity on the channel.
func waitForActivity(sub chan pingMsg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
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
			m.findPinger(msg.pkt.IPAddr.String()).history.AppendFirst(HistoryData{Seq: msg.pkt.Seq, isRecieved: false})
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
	for _, pinger := range m.pingerList {
		rows = append(rows, createRow(*pinger))
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
		fmt.Printf("error: %v", err)
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

func (p pinger) getLatestSuccess() {
	if len(p.history.Data) < 1 {
		return
	}
	history := p.history.Data
	if history[len(history)-1].isRecieved && history[len(history)-2].isRecieved {
		return
	}
	return
}
