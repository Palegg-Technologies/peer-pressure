package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

const TCPProtocolID = protocol.ID("tcp")
const FileProtocolID = protocol.ID("/file/1.0.0")

type oldNodeMenuModel struct {
	name    string
	cursor  int
	choices []string
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
			parent.Tabs = append(parent.Tabs, TabStyles[len(parent.Tabs)].Render(choice))

			switch choice {
			case "Send":
				// parent.state++
				sendFile(context.Background(), m.name, "share_me._test")

			case "Receive":
				// parent.state += 2
				receiveFile(context.Background(), m.name, "nodes/shared_1.txt")
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
	s += footerStyle.Render(footer)

	// Send the UI for rendering
	return s
}

func receiveFile(ctx context.Context, nodeName string, saveFilePath string) {
	nodeDir := filepath.Join("nodes", nodeName)
	prvBytes, _ := os.ReadFile(filepath.Join(nodeDir, "rsa.priv"))
	prvKey, _ := crypto.UnmarshalPrivateKey(prvBytes)

	// Load limiter config
	limiterCfg, err := os.Open("limitCfg.json")
	if err != nil {
		panic(err)
	}
	limiter, err := rcmgr.NewDefaultLimiterFromJSON(limiterCfg)
	if err != nil {
		panic(err)
	}
	rcm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		panic(err)
	}

	h, err := libp2p.New(libp2p.Identity(prvKey), libp2p.ResourceManager(rcm))
	if err != nil {
		panic(err)
	}
	defer h.Close()
	h.SetStreamHandler(TCPProtocolID, handleStream)

	f, err := os.Open(nodeDir)
	if err != nil {
		log.Panicln(err)
	}
	files, err := f.Readdir(0)
	if err != nil {
		log.Panicln(err)
	}

	topicNameFlag := "applesauce"
	for _, v := range files {
		if !v.IsDir() && v.Name() != "rsa.priv" && v.Name() != "rsa.pub" {
			topicNameFlag = v.Name()
		}
	}

	kademliaDHT := initDHT(ctx, h, nodeDir, topicNameFlag)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topicNameFlag)

	time.Sleep(time.Duration(1) * time.Minute)
	peerChan, err := routingDiscovery.FindPeers(ctx, topicNameFlag)
	if err != nil {
		panic(err)
	}
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

				stream, _ := h.NewStream(ctx, peer.ID, TCPProtocolID)
				rw := bufio.NewReader(stream)

				go readData(rw, saveFilePath)
			}
		}
		log.Printf("Receiver wait round: %d", i)
		time.Sleep(time.Duration(5) * time.Second)
		i++
	}
}

func readData(rw *bufio.Reader, saveFilePath string) {
	f, _ := os.Create(saveFilePath)
	defer f.Close()
	writer := bufio.NewWriter(f)
	for {
		str, err := rw.ReadByte()
		if err == io.EOF {
			log.Printf("%s done writing", saveFilePath)
			break
		} else if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}
		fmt.Println(str)
		err = writer.WriteByte(str)
		if err != nil {
			log.Println(err)
		}
	}
	writer.Flush()
}

func sendFile(ctx context.Context, nodeName string, sendFilePath string) {
	nodeDir := filepath.Join("nodes", nodeName)
	prvBytes, _ := os.ReadFile(filepath.Join(nodeDir, "rsa.priv"))
	prvKey, _ := crypto.UnmarshalPrivateKey(prvBytes)

	// Load limiter config
	limiterCfg, err := os.Open("limitCfg.json")
	if err != nil {
		panic(err)
	}
	limiter, err := rcmgr.NewDefaultLimiterFromJSON(limiterCfg)
	if err != nil {
		panic(err)
	}
	rcm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		panic(err)
	}

	h, err := libp2p.New(libp2p.Identity(prvKey), libp2p.ResourceManager(rcm))
	if err != nil {
		panic(err)
	}
	defer h.Close()

	f, err := os.Open(nodeDir)
	if err != nil {
		log.Panicln(err)
	}
	files, err := f.Readdir(0)
	if err != nil {
		log.Panicln(err)
	}

	topicNameFlag := "applesauce"
	for _, v := range files {
		if !v.IsDir() && v.Name() != "rsa.priv" && v.Name() != "rsa.pub" {
			topicNameFlag = v.Name()
		}
	}

	kademliaDHT := initDHT(ctx, h, nodeDir, topicNameFlag)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topicNameFlag)

	peerChan, err := routingDiscovery.FindPeers(ctx, topicNameFlag)
	if err != nil {
		panic(err)
	}
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
				writeData(rw, sendFilePath)
				stream.Close()
			}()
		}
	}
}

func handleStream(stream network.Stream) {

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReader(stream)

	go readData(rw, "xyz")

	// 'stream' will stay open until you close it (or the other side closes it).
}

func writeData(rw *bufio.Writer, sharedFilePath string) {
	f, _ := os.Open(sharedFilePath)
	stdReader := bufio.NewReader(f)
	defer f.Close()

	for {
		sendData, err := stdReader.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Error reading from stdin")
			panic(err)
		}

		log.Println(sendData)
		err = rw.WriteByte(sendData)
		if err != nil {
			log.Println("Error writing to buffer")
			panic(err)
		}
		log.Println(rw.Available())
		err = rw.Flush()
		if err != nil {
			log.Println("Error flushing buffer")
			panic(err)
		}
	}
}
