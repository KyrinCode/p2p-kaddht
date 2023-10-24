package main

import (
	"context"
	"fmt"
	"log"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

func Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, rendezvous string) {
	var routingDiscovery = drouting.NewRoutingDiscovery(dht)

	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)
			if err != nil {
				log.Fatal(err)
			}

			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				// err := h.Connect(ctx, p)
				// if err != nil {
				// 	fmt.Printf("Failed connecting to %s, error: %s\n", p.ID, err)
				// } else {
				// 	fmt.Printf("Connected to peer: %s\n", p.ID)
				// }
				if h.Network().Connectedness(p.ID) != network.Connected {
					_, err = h.Network().DialPeer(ctx, p.ID)
					if err != nil {
						fmt.Printf("Failed connecting to %s, error: %s\n", p.ID.String(), err)
						// err = h.Network().ClosePeer(p.ID)
						// if err != nil {
						// 	fmt.Printf("Failed disconnecting to %s, error: %s\n", p.ID.String(), err)
						// }
						continue
					} else {
						fmt.Printf("Connected to peer: %s\n", p.ID.String())
					}
				}
			}
		}
	}
}
