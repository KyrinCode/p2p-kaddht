// https://medium.com/rahasak/libp2p-pubsub-peer-discovery-with-kademlia-dht-c8b131550ac7

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	b58 "github.com/mr-tron/base58/base58" // test

	// test
	"github.com/multiformats/go-multiaddr"
)

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = time.Hour

// DiscoveryServiceTag is used in our mDNS advertisements to discover other peers.
const DiscoveryServiceTag = "room-pubsub"

func main() {
	ctx := context.Background()

	// create a new libp2p Host that listens on a random TCP port
	// we can specify port like /ip4/0.0.0.0/tcp/3326
	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		panic(err)
	}

	// view host details and addresses
	fmt.Printf("[Host ID] %s\n", host.ID().Pretty())
	fmt.Printf("Assigned listening addresses:\n")
	for _, addr := range host.Addrs() {
		fmt.Printf("%s\n", addr.String())
	}
	fmt.Printf("\n")

	// create a new PubSub service using the GossipSub router
	gossipSub, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// option 1
	// setup DHT with empty discovery peers
	// so this will be a discovery peer for others
	// this peer should run on cloud(with public ip address)
	// discoveryPeers := []multiaddr.Multiaddr{}

	// option 2
	// ipfs address of discovery peers
	// i have added one peer here, you could add multiple disovery peers
	multiAddr, err := multiaddr.NewMultiaddr("/ip4/10.249.1.47/tcp/54547/p2p/12D3KooWHvetCa5o8Xpz7YhnHqpGLnvmX3QtmhPrREiuFziUGPCm")
	if err != nil {
		panic(err)
	}
	// println(multiAddr.String())
	// setup DHT with discovery server
	// this peer could run behind the nat(with private ip address)
	discoveryPeers := []multiaddr.Multiaddr{multiAddr}

	dht, err := initDHT(ctx, host, discoveryPeers)
	if err != nil {
		panic(err)
	}

	// setup peer discovery
	go Discover(ctx, host, dht, "room")

	// setup local mDNS discovery
	// if err := setupDiscovery(host); err != nil {
	// 	panic(err)
	// }

	// join the pubsub topic called room
	room := "room"
	topic, err := gossipSub.Join(room)
	if err != nil {
		panic(err)
	}

	// subscribe to topic
	subscriber, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}
	stop := make(chan bool)
	go subscribe(subscriber, ctx, host.ID(), stop)

	// support unsubscribe
	// stop <- true
	// subscriber.Cancel()

	// test if bootstrap node help nodes to connect with each other
	time.Sleep(5 * time.Second)
	peerID, _ := b58.Decode("12D3KooWQAn3owZCKqU6ndPBgfuA3gyKntMe3rMoE7GqGGpBWpkL") // second node peer ID
	pID := peer.ID(string(peerID))
	println("?")
	if host.Network().Connectedness(pID) == network.Connected {
		fmt.Printf("Yes! Connected to peer %s\n", pID.String())
	}

	// create publisher
	publish(ctx, topic)
}

// start publisher to topic
func publish(ctx context.Context, topic *pubsub.Topic) {
	for {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Printf("Enter message to publish: \n")

			msg := scanner.Text()
			if len(msg) != 0 {
				// publish message to topic
				bytes := []byte(msg)
				topic.Publish(ctx, bytes)
			}
		}
	}
}

// start subsriber to topic
func subscribe(subscriber *pubsub.Subscription, ctx context.Context, hostID peer.ID, stop chan bool) {
	for {
		select {
		case <-stop:
			return
		default:
			msg, err := subscriber.Next(ctx)
			if err != nil {
				panic(err)
			}

			// only consider messages delivered by other peers
			if msg.ReceivedFrom == hostID {
				continue
			}

			fmt.Printf("Got message: %s, from: %s\n", string(msg.Data), msg.ReceivedFrom.String())
		}
	}
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("New peer discovered: %s\n", pi.ID.String())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("Error connecting to peer %s: %s\n", pi.ID.String(), err)
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h})
	return s.Start()
}
