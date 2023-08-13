package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/Azanul/peer-pressure/pkg/peer"
	"github.com/Azanul/peer-pressure/pkg/pressure/pb"
	"github.com/Azanul/peer-pressure/pkg/streamio"
	"github.com/Azanul/peer-pressure/tui/style"
	"github.com/charmbracelet/bubbles/filepicker"
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
	transfer   peer.Transfer
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
			parent.Tabs = append(parent.Tabs, style.TabStyles[len(parent.Tabs)](choice))

			switch choice {
			case "Send":
				parent.state++

			case "Receive":
				parent.state += 3
				err := receiveFile(context.Background(), m.name, m.transfer.EventCh, m.transfer.CommandCh)
				if err != nil {
					fmt.Println(style.ErrorTextStyle(err.Error()))
					cmds = append(cmds, tea.Quit)
				}
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
	s += style.FooterStyle(footer)

	// Send the UI for rendering
	return s
}

func receiveFile(ctx context.Context, nodeName string, eventCh chan peer.Event, cmdCh chan peer.Command) (err error) {
	p, err := peer.Load(nodeName)
	if err != nil {
		return
	}

	foundSender := false // flag for closing receiver

	h := p.Node
	h.SetStreamHandler(TCPProtocolID, func(stream network.Stream) {
		// Create a buffer stream for non blocking read and write.
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

		index := pb.Index{}
		err := pb.Read(rw.Reader, &index)
		if err != nil {
			log.Errorln(err)
			return
		}
		indexPath := "nodes/" + index.GetFilename() + ".ppindex"
		existingIndex, err := os.ReadFile(indexPath)
		if err == nil {
			log.Debugln("index file found, using existing index")
			err = proto.Unmarshal(existingIndex, &index)
			if err != nil {
				log.Errorln(err)
				return
			}
		} else {
			log.Warnln("index file not found, saving incoming index")
			index.Save()
		}

		cr := pb.ChunkRequest{
			Index: index.Progress + 1,
		}

		str := pb.Marshal(&cr)
		_, err = rw.Write(str)
		if err != nil {
			log.Errorln(err)
			return
		}
		log.Debugln(rw.Flush())

		dest := "nodes/" + index.GetFilename()
		f, err := os.OpenFile(dest, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if os.IsNotExist(err) {
			f, err = os.Create(dest)
		}
		if err != nil {
			log.Errorf("error opening/creating file %s: %v\n", dest, err)
			log.Errorln(err)
			return
		}

		streamio.StreamToFile(rw, f, eventCh, cmdCh)
		foundSender = true
	})

	peerChan, err := p.DiscoverPeers(ctx)
	if err != nil {
		return
	}

	log.Printf("R Peer ID: %s\n\n", h.ID())
	for i := 0; i < 30; i++ {
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue // No self connection
			}
			err := h.Connect(ctx, peer)
			if err != nil {
				log.Println("R Failed connecting to ", peer.ID.Pretty(), ", error:", err)
			} else {
				log.Println("R Connected to peer:", peer.ID.Pretty())
				foundSender = true
				break
			}
		}
		if foundSender {
			break
		}
		log.Printf("Receiver wait round: %d", i)
		time.Sleep(time.Duration(5) * time.Second)
	}
	return
}

func sendFile(ctx context.Context, nodeName string, sendFilePath string, eventCh chan peer.Event, cmdCh chan peer.Command) (err error) {
	p, err := peer.Load(nodeName)
	if err != nil {
		return
	}

	peerChan, err := p.DiscoverPeers(ctx)
	if err != nil {
		return
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
				return err
			}
			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			go func() {
				f, err := os.Open(sendFilePath)
				if err != nil {
					fmt.Println(style.ErrorTextStyle(err.Error()))
					return
				}
				streamio.FileToStream(rw, f, eventCh, cmdCh)
				stream.Close()
			}()
		}
	}
	return
}
