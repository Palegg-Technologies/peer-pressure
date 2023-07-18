package peer

import (
	"bufio"
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/Azanul/peer-pressure/pkg/util"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
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

	return Peer{
		Node:       node,
		Name:       name,
		rendezvous: rendezvous,
		privKey:    prvKey,
		PubKey:     pubKey,
		peerDir:    filepath.Join(".", "nodes", name),
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

func (p *Peer) DiscoverPeers(ctx context.Context, peerDir string) {
	kademliaDHT := p.initDHT(ctx, peerDir)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, p.rendezvous)

	file, err := os.OpenFile(filepath.Join(peerDir, p.rendezvous), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	// Look for others who have announced and attempt to connect to them
	// Save connected peers to connect
	log.Println("Searching for peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, p.rendezvous)
	if err != nil {
		panic(err)
	}
	for peer := range peerChan {
		if peer.ID == p.Node.ID() {
			continue // No self connection
		}
		err := p.Node.Connect(ctx, peer)
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
	log.Println(dht.DefaultBootstrapPeers)
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func(addr multiaddr.Multiaddr) {
			defer wg.Done()
			if err := p.Node.Connect(ctx, *peerinfo); err != nil {
				log.Println("Bootstrap warning:", err)
			} else {
				log.Println("Connection established with bootstrap node:", *peerinfo)
				writer.WriteString(addr.String())
			}
		}(peerAddr)
	}
	wg.Wait()

	return kademliaDHT
}

func (p *Peer) GetPeerDir() string {
	return p.peerDir
}
