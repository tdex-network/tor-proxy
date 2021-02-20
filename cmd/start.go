package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tdex-network/tor-proxy/pkg/torproxy"
	"github.com/urfave/cli/v2"
)

var start = cli.Command{
	Name:  "start",
	Usage: "start the reverse proxy",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "registry",
			Usage:    "JSON file or string with list of onion endpoints. For more info see https://github.com/TDex-network/tdex-registry",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "domain",
			Usage: "TLD domain to expose the reverse proxy",
		},
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "expose in plaintext in localhost",
			Value: true,
		},
		&cli.IntFlag{
			Name:  "port",
			Usage: "listening port for the reverse proxy",
			Value: 7070,
		},
		&cli.StringFlag{
			Name:  "socks5_hostname",
			Usage: "the socks5 hostname exposed by the tor client",
			Value: "127.0.0.1",
		},
		&cli.IntFlag{
			Name:  "socks5_port",
			Usage: "the socks5 port exposed by the tor client",
			Value: 9150,
		},
	},
	Action: startAction,
}

func startAction(ctx *cli.Context) error {

	// will check if domain is given, now we default to insecure
	listeningAddress := "localhost:" + fmt.Sprint(ctx.Int("port"))

	// registry
	var data []map[string]string
	err := json.Unmarshal([]byte(ctx.String("registry")), &data)
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	redirects := make([]string, 0)
	for _, v := range data {
		if strings.Contains(v["endpoint"], "onion") {
			redirects = append(redirects, v["endpoint"])
		}
	}
	if len(redirects) == 0 {
		return fmt.Errorf("no onion endpoints found in registry")
	}

	// Serve the reverse proxy
	proxy, _ := torproxy.NewTorProxyFromHostAndPort(
		ctx.String("socks5_hostname"),
		ctx.Int("socks5_port"),
	)
	defer proxy.Close()

	// Add redirects array
	proxy.WithRedirects(redirects)

	log.Printf("Serving tor proxy on %s\n", listeningAddress)
	if err := proxy.Serve(listeningAddress); err != nil {
		return fmt.Errorf("serving proxy: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	return nil
}
