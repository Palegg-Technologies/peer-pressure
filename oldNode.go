package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Azanul/peer-pressure/pkg/peer"
	"github.com/Azanul/peer-pressure/pkg/util"
	"github.com/Azanul/peer-pressure/tui"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const TCPProtocolID = protocol.ID("tcp")
const FileProtocolID = protocol.ID("/file/1.0.0")

type oldNodeMenuModel struct {
	name       string
	cursor     int
	choices    []string
	filepicker filepicker.Model
	progress   progress.Model
}

func (m *oldNodeMenuModel) Update(parent *model, msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q", "esc":
			return parent, tea.Quit

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
			if len(parent.Tabs) > 1 {
				parent.choices = TabChoices[TabToI[parent.Tabs[len(parent.Tabs)-1]]]
				parent.Tabs = parent.Tabs[:len(parent.Tabs)-1]
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			choice := m.choices[m.cursor]
			parent.Tabs = append(parent.Tabs, tui.TabStyles[len(parent.Tabs)].Render(choice))

			switch choice {
			case "Send":
				parent.state++
				// sendFile(context.Background(), m.name, m.filepicker.FileSelected)

			case "Receive":
				// parent.state += 2
				receiveFile(context.Background(), m.name)
			}

		}
	}

	return parent, tea.Batch(cmds...)
}

func (m oldNodeMenuModel) View() string {
	s := "\n\n"

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
	s += mainChoices

	// The footer
	footer := ""
	s += tui.FooterStyle.Render(footer)

	// Send the UI for rendering
	return s
}

func receiveFile(ctx context.Context, nodeName string) {
	p := peer.Load(nodeName)

	h := p.Node
	h.SetStreamHandler(TCPProtocolID, func(stream network.Stream) {
		// Create a buffer stream for non blocking read and write.
		rw := bufio.NewReader(stream)

		generatedName := fmt.Sprintf("nodes/%s.part", util.RandString(10))
		f, err := os.Create(generatedName)
		if err != nil {
			log.Println(err)
			return
		}

		receivedName := util.StreamToFile(rw, f)
		log.Println(receivedName)
		if receivedName != "" {
			receivedName = filepath.Base(receivedName)
			err = os.Rename(generatedName, "nodes/"+receivedName)
			if err != nil {
				log.Println(err)
			}
		}
	})

	peerChan, err := p.DiscoverPeers(ctx)
	if err != nil {
		panic(err)
	}

	log.Printf("R Peer ID: %s\n\n", h.ID())
	i := 0
	for {
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue // No self connection
			}
			err := h.Connect(ctx, peer)
			if err != nil {
				log.Println("R Failed connecting to ", peer.ID.Pretty(), ", error:", err)
			} else {
				log.Println("R Connected to peer:", peer.ID.Pretty())
			}
		}
		log.Printf("Receiver wait round: %d", i)
		time.Sleep(time.Duration(5) * time.Second)
		i++
	}
}

func sendFile(ctx context.Context, nodeName string, sendFilePath string) {
	p := peer.Load(nodeName)

	peerChan, err := p.DiscoverPeers(ctx)
	if err != nil {
		panic(err)
	}

	h := p.Node
	log.Printf("S Peer ID: %s\n\n", h.ID())
	for peer := range peerChan {
		if peer.ID == h.ID() {
			continue // No self connection
		}
		err := h.Connect(ctx, peer)
		if err != nil {
			log.Println("S Failed connecting to ", peer.ID.Pretty(), ", error:", err)
		} else {
			log.Println("S Connected to:", peer.ID.Pretty())
			stream, err := h.NewStream(ctx, peer.ID, TCPProtocolID)
			if err != nil {
				log.Panicln(err)
			}
			rw := bufio.NewWriter(stream)

			go func() {
				f, err := os.Open(sendFilePath)
				if err != nil {
					log.Println(err)
					return
				}
				util.FileToStream(rw, f)
				stream.Close()
			}()
		}
	}
}
