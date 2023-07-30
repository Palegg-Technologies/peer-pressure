package peer

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sync"

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

func New(name, rendezvous string) Peer {
	if rendezvous == "" {
		rendezvous = "applesauce"
	}

	// Creates a new RSA key pair for this host.
	prvKey, pubKey, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		log.Panicln(err)
	}

	// start a libp2p host with default settings
	h, err := libp2p.New(libp2p.Identity(prvKey), libp2p.ResourceManager(loadReasourceManager()))
	if err != nil {
		panic(err)
	}
	defer h.Close()

	log.Println(h.ID())
	log.Println(h.Addrs())

	return Peer{
		Node:       h,
		Name:       name,
		rendezvous: rendezvous,
		privKey:    prvKey,
		PubKey:     pubKey,
		peerDir:    filepath.Join("nodes", name),
	}
}

func Load(name string) Peer {
	nodeDir := filepath.Join("nodes", name)
	prvBytes, _ := os.ReadFile(filepath.Join(nodeDir, "rsa.priv"))
	prvKey, _ := crypto.UnmarshalPrivateKey(prvBytes)
	pubBytes, _ := os.ReadFile(filepath.Join(nodeDir, "rsa.pub"))
	pubKey, _ := crypto.UnmarshalPublicKey(pubBytes)

	h, err := libp2p.New(libp2p.Identity(prvKey), libp2p.ResourceManager(loadReasourceManager()))
	if err != nil {
		panic(err)
	}

	f, err := os.Open(nodeDir)
	if err != nil {
		log.Panicln(err)
	}
	files, err := f.Readdir(0)
	if err != nil {
		log.Panicln(err)
	}

	rendezvous := "applesauce"
	for _, v := range files {
		if !v.IsDir() && v.Name() != "rsa.priv" && v.Name() != "rsa.pub" {
			rendezvous = v.Name()
		}
	}

	return Peer{
		Node:       h,
		Name:       name,
		rendezvous: rendezvous,
		privKey:    prvKey,
		PubKey:     pubKey,
		peerDir:    filepath.Join("nodes", name),
	}
}

func (p *Peer) Save() {
	// make directory for node info
	err := os.MkdirAll(p.peerDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	// write private key
	privBytes, err := crypto.MarshalPrivateKey(p.privKey)
	if err != nil {
		panic(err)
	}
	util.AppendStringToFile(filepath.Join(p.peerDir, "rsa.priv"), string(privBytes))

	// write public key
	pubBytes, err := crypto.MarshalPublicKey(p.PubKey)
	if err != nil {
		panic(err)
	}
	util.AppendStringToFile(filepath.Join(p.peerDir, "rsa.pub"), string(pubBytes))
}

func (p *Peer) DiscoverPeers(ctx context.Context) (<-chan peer.AddrInfo, error) {
	kademliaDHT := p.initDHT(ctx, p.peerDir)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, p.rendezvous)

	return routingDiscovery.FindPeers(ctx, p.rendezvous)
}

func (p *Peer) initDHT(ctx context.Context, peerDir string) *dht.IpfsDHT {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, p.Node)
	if err != nil {
		panic(err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	file, err := os.OpenFile(filepath.Join(peerDir, p.rendezvous), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Panicln(err)
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

	return kademliaDHT
}

func (p *Peer) GetPeerDir() string {
	return p.peerDir
}

func (p *Peer) GetRendezvous() string {
	return p.rendezvous
}

func loadReasourceManager() network.ResourceManager {
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
	return rcm
}
