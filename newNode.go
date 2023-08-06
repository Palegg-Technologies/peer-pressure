package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/Azanul/peer-pressure/pkg/peer"
	"github.com/Azanul/peer-pressure/tui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type createFormModel struct {
	inputs  []textinput.Model
	focused int
}

func (m *createFormModel) Update(parent *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, 2)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			name := m.inputs[0].Value()
			createNewNode(name, m.inputs[1].Value())
			newChoice := name
			parent.Tabs = parent.Tabs[:1]
			parent.state = 0
			TabChoices[0] = append(TabChoices[0], newChoice)
			parent.choices = TabChoices[0]
			return parent, nil

		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyCtrlQ:
			return parent, tea.Quit

		case tea.KeyCtrlLeft:
			parent.state--
			parent.Tabs = parent.Tabs[:len(parent.Tabs)-1]
			return parent, nil

		case tea.KeyShiftTab, tea.KeyCtrlP:
			m.prevInput()

		case tea.KeyTab, tea.KeyCtrlN:
			m.nextInput()
		}

		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.inputs[m.focused].Focus()
	}

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return parent, tea.Batch(cmds...)
}

// nextInput focuses the next input field
func (m *createFormModel) nextInput() {
	m.focused = (m.focused + 1) % len(m.inputs)
}

// prevInput focuses the previous input field
func (m *createFormModel) prevInput() {
	m.focused--
	// Wrap around
	if m.focused < 0 {
		m.focused = len(m.inputs) - 1
	}
}

func (m createFormModel) View() string {
	// The footer
	footer := "\nPress Ctrl+◀  to go back"
	footer += "\nPress esc / Ctrl+q to quit.\n"

	return fmt.Sprintf(
		`
 %s
 %s
 %s
 %s

 %s
`,
		tui.NNInputStyle.Width(30).Render("Name"),
		m.inputs[0].View(),
		tui.NNInputStyle.Width(30).Render("Rendezvous"),
		m.inputs[1].View(),
		tui.NNContinueStyle.Render("Continue ->"),
	) + "\n" + tui.FooterStyle.Render(footer)
}

func createNewNode(name string, rendezvous string) {
	p, err := peer.New(name, rendezvous)
	if err != nil {
		panic(err)
	}
	p.Save()
	go func() {
		file, err := os.OpenFile(filepath.Join(p.GetPeerDir(), p.GetRendezvous()), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			log.Panicln(err)
		}
		defer file.Close()
		writer := bufio.NewWriter(file)

		peerChan, err := p.DiscoverPeers(context.TODO())
		if err != nil {
			panic(err)
		}

		for peer := range peerChan {
			if peer.ID == p.Node.ID() {
				continue // No self connection
			}
			err := p.Node.Connect(context.Background(), peer)
			if err != nil {
				log.Println("Failed connecting to ", peer.ID.Pretty(), ", error:", err)
			} else {
				log.Println("Connected to:", peer.ID.Pretty())
				for _, addr := range peer.Addrs {
					writer.WriteString(addr.String())
				}
			}
		}
	}()
}
