//go:build android && !cgo
// +build android,!cgo

package main

import (
	"context"
	"net"
	"sync"
	"time"
)

func init() {
	// On Android, when not using cgo, we need to manually set up the default DNS resolver.
	// This resolver will attempt Cloudflare's DNS over both IPv4 and IPv6.

	var dialer net.Dialer
	dnsServers := []string{
		"[2a10:50c0::ad1:ff]:53", // Cloudflare IPv6
		"[2a10:50c0::ad2:ff]:53", // Cloudflare IPv6
		"94.140.14.14:53",                // Cloudflare IPv4
		"94.140.15.15:53",                // Cloudflare IPv4
	}

	net.DefaultResolver = &net.Resolver{
		PreferGo: false,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			var wg sync.WaitGroup
			result := make(chan net.Conn, 1)
			errChan := make(chan error, len(dnsServers))

			for _, ip := range dnsServers {
				wg.Add(1)
				go func(ip string) {
					defer wg.Done()
					conn, err := dialer.DialContext(ctx, "udp", ip)
					if err == nil {
						select {
						case result <- conn:
							cancel()
						default:
						}
					} else {
						errChan <- err
					}
				}(ip)
			}

			go func() {
				wg.Wait()
				close(result)
				close(errChan)
			}()

			select {
			case conn := <-result:
				return conn, nil
			case <-time.After(2 * time.Second):
				return nil, net.ErrClosed
			}
		},
	}
}
