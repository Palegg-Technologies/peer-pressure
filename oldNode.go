package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Azanul/peer-pressure/pkg/util"
	"github.com/Azanul/peer-pressure/tui"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
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
	h.SetStreamHandler(TCPProtocolID, func(stream network.Stream) {
		// Create a buffer stream for non blocking read and write.
		rw := bufio.NewReader(stream)

		go util.ReadFromStream(rw, "nodes/saveFilePath")

		// 'stream' will stay open until you close it (or the other side closes it).
	})

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
			}
		}
		log.Printf("Receiver wait round: %d", i)
		time.Sleep(time.Duration(5) * time.Second)
		i++
	}
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
				util.WriteToStream(rw, sendFilePath)
				stream.Close()
			}()
		}
	}
}

func initDHT(ctx context.Context, h host.Host, peerDir string, topicNameFlag string) *dht.IpfsDHT {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		panic(err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	file, err := os.OpenFile(filepath.Join(peerDir, topicNameFlag), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	var wg sync.WaitGroup
	log.Println(dht.DefaultBootstrapPeers)
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerinfo); err != nil {
				log.Println("Bootstrap warning:", err)
			} else {
				log.Println("Connection established with bootstrap node:", *peerinfo)
				writer.WriteString(peerAddr.String())
			}
		}()
	}
	wg.Wait()

	return kademliaDHT
}
