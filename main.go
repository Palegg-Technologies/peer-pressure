package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azanul/peer-pressure/tui"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	TabToI = map[string]int{
		"Main Menu":       0,
		"Create new node": 1,
	}

	TabChoices = [][]string{
		{},
		{"Send", "Receive"},
	}

	nodeCreate = createFormModel{
		inputs: []textinput.Model{
			textinput.New(),
			textinput.New(),
		},
	}

	crrNode = oldNodeMenuModel{
		name:       "test",
		choices:    []string{"Send", "Receive"},
		filepicker: filepicker.New(),
		progress:   progress.New(progress.WithDefaultGradient()),
	}
)

type sessionState uint

const (
	mainMenu sessionState = iota
	newNodeForm
	oldNodeMenu
	sendFileExplorer
	sendLoader
	receiveFileMenu
)

type model struct {
	state   sessionState // state to track which model is focussed
	Tabs    []string
	choices []string // options [create new node, list of nodes...]
	cursor  int      // which option our cursor is pointing at
}

func (m model) Init() tea.Cmd {
	return crrNode.filepicker.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		crrNode.filepicker, cmd = crrNode.filepicker.Update(msg)
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		progressModel, cmd := crrNode.progress.Update(msg)
		crrNode.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	}

	switch m.state {
	case newNodeForm:
		return nodeCreate.Update(m, msg, m.cursor)

	case oldNodeMenu:
		return crrNode.Update(m, msg)

	case sendFileExplorer:
		crrNode.filepicker, cmd = crrNode.filepicker.Update(msg)

		// Did the user select a file?
		if didSelect, path := crrNode.filepicker.DidSelectFile(msg); didSelect {
			sendFile(context.TODO(), crrNode.name, path)
			m.state++
		}
		return m, cmd

	case sendLoader:
		if crrNode.progress.Percent() == 1 {
			m.state = 0
			crrNode.progress.SetPercent(0)
		} else {
			cmds = append(cmds, crrNode.progress.IncrPercent(0.01))
		}

	default:
		switch msg := msg.(type) {

		// Is it a key press?
		case tea.KeyMsg:

			// Cool, what was the actual key pressed?
			switch msg.String() {

			// These keys should exit the program.
			case "ctrl+c", "q", "esc":
				return m, tea.Quit

			// The "up" and "k" keys move the cursor up
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			// The "down" and "j" keys move the cursor down
			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}

			case "left", "backspace":
				if len(m.Tabs) > 1 {
					m.choices = TabChoices[TabToI[m.Tabs[len(m.Tabs)-1]]]
					m.Tabs = m.Tabs[:len(m.Tabs)-1]
				}

			// The "enter" key and the spacebar (a literal space) toggle
			// the selected state for the item that the cursor is pointing at.
			case "enter", " ":
				choice := m.choices[m.cursor]
				m.Tabs = append(m.Tabs, tui.TabStyles[len(m.Tabs)].Render(choice))

				if choice == "Create new node" {
					m.state++
					nodeCreate.inputs[0].Focus()
				} else {
					m.state += 2
					crrNode.name = choice
				}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	// Tabs
	s := lipgloss.JoinHorizontal(lipgloss.Top, m.Tabs...)

	switch m.state {
	case mainMenu:
		// The header
		header := tui.HeaderStyle.Render("Create new node or select an already existing one\n")

		mainChoices := ""
		// Iterate over our choices
		for i, choice := range m.choices {

			// Is the cursor pointing at this choice?
			cursor := " " // no cursor
			if m.cursor == i {
				cursor = ">" // cursor!
			}

			// Render the row
			mainChoices += fmt.Sprintf("%s %s\n", cursor, choice)
		}

		// Tab header
		s = lipgloss.JoinVertical(lipgloss.Left, s, header, mainChoices)

		// The footer
		footer := ""
		if len(m.Tabs) > 1 {
			footer = "\nPress ◀ / Backspace to go back"
		}
		footer += "\nPress q to quit.\n"
		s += tui.FooterStyle.Render(footer)

	case newNodeForm:
		s += nodeCreate.View()

	case oldNodeMenu:
		s += crrNode.View()

	case sendFileExplorer:
		s += "\n\n" + crrNode.filepicker.View()
		log.Println(crrNode.filepicker.CurrentDirectory, crrNode.filepicker.AutoHeight)

	case sendLoader:
		s += "\n\n" + crrNode.progress.View()

	case receiveFileMenu:
		tea.Println("Not yet implemented")
	}

	// Send the UI for rendering
	return s
}

func initialModel() model {
	choices := []string{"Create new node"}

	crrNode.filepicker.CurrentDirectory, _ = os.UserHomeDir()

	err := os.MkdirAll("./nodes", os.ModePerm)
	if err != nil {
		panic(err)
	}

	f, err := os.Open("./nodes")
	if err != nil {
		log.Panicln(err)
	}
	files, err := f.Readdir(0)
	if err != nil {
		log.Panicln(err)
	}

	for _, v := range files {
		if v.IsDir() {
			choices = append(choices, v.Name())
		}
	}

	TabChoices[0] = choices

	mainModel := model{
		choices: choices,
		Tabs:    []string{tui.TabStyles[0].Render("Main Menu")},
	}
	return mainModel
}

func main() {
	f, err := os.OpenFile("log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// starting our program
	m := initialModel()
	if _, err := tea.NewProgram(&m).Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
