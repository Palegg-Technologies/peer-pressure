package peer

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	log "github.com/sirupsen/logrus"

	"github.com/Azanul/peer-pressure/pkg/util"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
)

type Peer struct {
	Node       host.Host
	Name       string
	rendezvous string
	peerDir    string
	privKey    crypto.PrivKey
	crypto.PubKey
}

func New(name, rendezvous string) (*Peer, error) {
	if rendezvous == "" {
		rendezvous = "applesauce"
	}

	// Creates a new RSA key pair for this host.
	prvKey, pubKey, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return nil, err
	}

	// start a libp2p host with default settings
	h, err := libp2p.New(libp2p.Identity(prvKey), libp2p.ResourceManager(loadResourceManager()))
	if err != nil {
		return nil, err
	}
	defer h.Close()

	log.Println(h.ID())
	log.Println(h.Addrs())

	return &Peer{
		Node:       h,
		Name:       name,
		rendezvous: rendezvous,
		privKey:    prvKey,
		PubKey:     pubKey,
		peerDir:    filepath.Join("nodes", name),
	}, nil
}

func Load(name string) (*Peer, error) {
	nodeDir := filepath.Join("nodes", name)
	prvBytes, _ := os.ReadFile(filepath.Join(nodeDir, "rsa.priv"))
	prvKey, _ := crypto.UnmarshalPrivateKey(prvBytes)
	pubBytes, _ := os.ReadFile(filepath.Join(nodeDir, "rsa.pub"))
	pubKey, _ := crypto.UnmarshalPublicKey(pubBytes)

	h, err := libp2p.New(libp2p.Identity(prvKey), libp2p.ResourceManager(loadResourceManager()))
	if err != nil {
		return nil, err
	}

	f, err := os.Open(nodeDir)
	if err != nil {
		return nil, err
	}
	files, err := f.Readdir(0)
	if err != nil {
		return nil, err
	}

	rendezvous := "applesauce"
	for _, v := range files {
		if !v.IsDir() && v.Name() != "rsa.priv" && v.Name() != "rsa.pub" {
			rendezvous = v.Name()
		}
	}

	return &Peer{
		Node:       h,
		Name:       name,
		rendezvous: rendezvous,
		privKey:    prvKey,
		PubKey:     pubKey,
		peerDir:    filepath.Join("nodes", name),
	}, nil
}

func (p *Peer) Save() (err error) {
	// make directory for node info
	err = os.MkdirAll(p.peerDir, os.ModePerm)
	if err != nil {
		return
	}

	// write private key
	privBytes, err := crypto.MarshalPrivateKey(p.privKey)
	if err != nil {
		return
	}
	err = util.AppendStringToFile(filepath.Join(p.peerDir, "rsa.priv"), string(privBytes))
	if err != nil {
		return
	}

	// write public key
	pubBytes, err := crypto.MarshalPublicKey(p.PubKey)
	if err != nil {
		return
	}
	return util.AppendStringToFile(filepath.Join(p.peerDir, "rsa.pub"), string(pubBytes))
}

func (p *Peer) DiscoverPeers(ctx context.Context) (<-chan peer.AddrInfo, error) {
	kademliaDHT, err := p.initDHT(ctx, p.peerDir)
	if err != nil {
		return nil, err
	}
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, p.rendezvous)

	return routingDiscovery.FindPeers(ctx, p.rendezvous)
}

func (p *Peer) initDHT(ctx context.Context, peerDir string) (*dht.IpfsDHT, error) {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, p.Node)
	if err != nil {
		return nil, err
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filepath.Join(peerDir, p.rendezvous), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func(addr multiaddr.Multiaddr, pInfo *peer.AddrInfo) {
			defer wg.Done()
			if err := p.Node.Connect(ctx, *pInfo); err != nil {
				log.Printf("Bootstraping %v warning: %v\n", *pInfo, err)
			} else {
				log.Println("Connection established with bootstrap node:", *pInfo)
				writer.WriteString(addr.String())
			}
		}(peerAddr, peerinfo)
	}
	wg.Wait()

	return kademliaDHT, nil
}

func (p *Peer) GetPeerDir() string {
	return p.peerDir
}

func (p *Peer) GetRendezvous() string {
	return p.rendezvous
}

type SignalType int8

const (
	Pause Command = iota
	Continue
	Stop
)

type Event struct {
	Type SignalType
	Data interface{}
}

type Command SignalType

type transferState int8

const (
	active transferState = iota
	paused
	inactive
)

type Transfer struct {
	state     transferState
	Progress  progress.Model
	EventCh   chan Event
	CommandCh chan Command
	TempPerc  float64
}

func (t *Transfer) Paused() bool {
	return t.state == paused
}
func (t *Transfer) Pause() {
	t.CommandCh <- Pause
	t.state = paused
}

func (t *Transfer) Continue() {
	t.CommandCh <- Continue
	t.state = active
}

func (t *Transfer) Toggle() {
	if t.state == paused {
		t.Continue()
	} else {
		t.Pause()
	}
}

func (t *Transfer) Stop() {
	t.CommandCh <- Continue
	t.state = inactive
}

func loadResourceManager() network.ResourceManager {
	limiterCfg, err := os.Open("limitCfg.json")
	if err != nil {
		if os.IsNotExist(err) {
			defaultConfig := getDefaultLimiter()
			err := saveLimiterConfig(defaultConfig)
			if err != nil {
				log.Errorf("Error creating and saving default limiter config: %s", err)
				return nil
			}
			limiterCfg, err = os.Open("limitCfg.json")
			if err != nil {
				log.Errorf("Error opening 'limitCfg.json' after creating: %s", err)
				return nil
			}
		} else {
			log.Errorf("Error opening 'limitCfg.json': %s", err)
			return nil
		}
	}
	defer limiterCfg.Close()

	limiter, err := rcmgr.NewDefaultLimiterFromJSON(limiterCfg)
	if err != nil {
		log.Errorf("Error parsing limiter config from JSON: %s", err)
		return nil
	}
	rcm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		log.Errorf("Error creating ResourceManager: %s", err)
		return nil
	}
	return rcm
}

func getDefaultLimiter() *rcmgr.Limiter {
	defaultLimitConfig := `
	{
		"System": {
		  "StreamsInbound": 4096,
		  "StreamsOutbound": 32768,
		  "Conns": 64000,
		  "ConnsInbound": 512,
		  "ConnsOutbound": 32768,
		  "FD": 64000
		},
		"Transient": {
		  "StreamsInbound": 4096,
		  "StreamsOutbound": 32768,
		  "ConnsInbound": 512,
		  "ConnsOutbound": 32768,
		  "FD": 64000
		},
		"ProtocolDefault": {
		  "StreamsInbound": 1024,
		  "StreamsOutbound": 32768
		},
		"ServiceDefault": {
		  "StreamsInbound": 2048,
		  "StreamsOutbound": 32768
		}
	  }`

	drcm, _ := rcmgr.NewDefaultLimiterFromJSON(strings.NewReader(defaultLimitConfig))
	return &drcm
}

func saveLimiterConfig(config *rcmgr.Limiter) error {
	file, err := os.Create("limitCfg.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(config)
}
