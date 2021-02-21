package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	registryBytes, err := getRegistryJSON(ctx.String("registry"))
	if err != nil {
		return fmt.Errorf("laoding json: %w", err)
	}

	redirects, err := registryJSONToRedirects(registryBytes)
	if err != nil {
		return fmt.Errorf("validating json: %w", err)
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

	// Catch SIGTERM and SIGINT signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	return nil
}

// getRegistryJSON will check if the given string is a) a JSON by itself b) if is a path to a file c) remote url
func getRegistryJSON(source string) ([]byte, error) {

	// check if it is a json the given source already
	if isArrayOfObjectsJSON(source) {
		return []byte(source), nil
	}

	// check if is a valid URL
	if isValidURL(source) {
		return fetchFromRemoteURL(source)
	}

	// in the end check if is a path to a file. If it exists try to read
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		return fetchFromFilePath(source)
	}

	return nil, errors.New("source must be either a valid JSON string, a remote URL or a valid path to a JSON file")
}
