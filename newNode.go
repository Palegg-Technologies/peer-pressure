package main

import (
	"context"
	"fmt"

	"github.com/Azanul/peer-pressure/pkg/peer"
	"github.com/Azanul/peer-pressure/tui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type createFormModel struct {
	inputs  []textinput.Model
	focused int
}

func (m *createFormModel) Update(parent *model, msg tea.Msg, option int) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, 2)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			name := m.inputs[0].Value()
			createNewNode(name, m.inputs[1].Value(), option)
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

func createNewNode(name string, rendezvous string, opt int) {
	nn := peer.New(name, rendezvous)
	nn.Save()
	go nn.DiscoverPeers(context.TODO(), nn.GetPeerDir())
}
