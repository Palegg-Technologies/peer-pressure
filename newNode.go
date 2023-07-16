package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
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
			newChoice := createNewNode(m.inputs[0].Value(), m.inputs[1].Value(), option)
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
		inputStyle.Width(30).Render("Name"),
		m.inputs[0].View(),
		inputStyle.Width(30).Render("Rendezvous"),
		m.inputs[1].View(),
		continueStyle.Render("Continue ->"),
	) + "\n" + footerStyle.Render(footer)
}

func createNewNode(name string, rendezvous string, opt int) string {
	if rendezvous == "" {
		rendezvous = "applesauce"
	}

	// Creates a new RSA key pair for this host.
	prvKey, pubKey, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		log.Panicln(err)
	}

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

	// start a libp2p node with default settings
	node, err := libp2p.New(libp2p.Identity(prvKey), libp2p.ResourceManager(rcm))
	if err != nil {
		panic(err)
	}
	defer node.Close()

	log.Println(node.ID())
	log.Println(node.Addrs())

	peerDir := filepath.Join(".", "nodes", name)

	log.Println(peerDir)
	// make directory for node info
	err = os.MkdirAll(peerDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	// write private key
	privBytes, err := crypto.MarshalPrivateKey(prvKey)
	if err != nil {
		panic(err)
	}
	appendStringToFile(filepath.Join(peerDir, "rsa.priv"), string(privBytes))

	// write public key
	pubBytes, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		panic(err)
	}
	appendStringToFile(filepath.Join(peerDir, "rsa.pub"), string(pubBytes))

	// discover peers
	go discoverPeers(context.Background(), node, peerDir, rendezvous)

	return name
}

func appendStringToFile(path string, content string) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(content)
	if err != nil {
		log.Panicln(err)
	}

	err = writer.Flush()
	if err != nil {
		log.Fatalln(err)
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

func discoverPeers(ctx context.Context, h host.Host, peerDir string, topicNameFlag string) {
	kademliaDHT := initDHT(ctx, h, peerDir, topicNameFlag)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topicNameFlag)

	file, err := os.OpenFile(filepath.Join(peerDir, topicNameFlag), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	// Look for others who have announced and attempt to connect to them
	// Save connected peers to connect
	log.Println("Searching for peers...")
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
			log.Println("Failed connecting to ", peer.ID.Pretty(), ", error:", err)
		} else {
			log.Println("Connected to:", peer.ID.Pretty())
			for _, addr := range peer.Addrs {
				writer.WriteString(addr.String())
			}
		}
	}

	log.Println("Peer discovery complete")
}
