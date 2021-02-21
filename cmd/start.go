package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/tdex-network/tor-proxy/pkg/torproxy"
	"github.com/urfave/cli/v2"
	"github.com/weppos/publicsuffix-go/publicsuffix"
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
			Usage: "TLD domain to obtain and renew the SSL certificate expose the reverse proxy",
		},
		&cli.StringFlag{
			Name:  "email",
			Usage: "email address to signify agreement and to be notified in case of issues with SSL certificate",
		},
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "expose in plaintext in localhost",
			Value: false,
		},
		&cli.IntFlag{
			Name:  "port",
			Usage: "listening port for the reverse proxy",
			Value: 7070,
		},
		&cli.BoolFlag{
			Name:  "use-tor",
			Usage: "use embedded tor client",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "socks5-hostname",
			Usage: "the socks5 hostname exposed by the tor client",
			Value: "127.0.0.1",
		},
		&cli.IntFlag{
			Name:  "socks5-port",
			Usage: "the socks5 port exposed by the tor client",
			Value: 9050,
		},
	},
	Action: startAction,
}

func startAction(ctx *cli.Context) error {

	// load registry json
	registryBytes, err := getRegistryJSON(ctx.String("registry"))
	if err != nil {
		return fmt.Errorf("laoding json: %w", err)
	}

	// parse registry json
	redirects, err := registryJSONToRedirects(registryBytes)
	if err != nil {
		return fmt.Errorf("validating json: %w", err)
	}

	var proxy *torproxy.TorProxy
	if ctx.Bool("use-tor") {
		// use the embedded tor client and expose it on :9050
		proxy, err = torproxy.NewTorProxy()
	} else {
		// use an external socks5 interface
		proxy, err = torproxy.NewTorProxyFromHostAndPort(
			ctx.String("socks5-hostname"),
			ctx.Int("socks5-port"),
		)
	}
	if err != nil {
		return fmt.Errorf("creating tor instance: %w", err)
	}

	// Add redirects to the proxy
	proxy.WithRedirects(redirects)

	// check if insecure flag, otherwise domain MUST be present to obtain SSL certificate
	var address string
	var tlsOptions *torproxy.TLSOptions
	if ctx.Bool("insecure") {
		address = ":" + fmt.Sprint(ctx.Int("port"))
	} else {
		email := ctx.String("email")
		domain := ctx.String("domain")

		// check if given domain is valid URL
		if len(domain) == 0 || !isValidDomain(domain) {
			return errors.New("domain is not a valid url to request a SSL certificate. Do you want to use --insecure?")
		}

		address = ":443"
		tlsOptions = &torproxy.TLSOptions{
			Domains: []string{domain},
			Email:   email,
		}
	}

	log.Printf("Serving tor proxy on %s\n", address)

	if err := proxy.Serve(address, tlsOptions); err != nil {
		return fmt.Errorf("serving proxy: %w", err)
	}
	defer proxy.Listener.Close()

	// Catch SIGTERM and SIGINT signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	fmt.Println("Shutdown")

	return nil
}

func isValidURL(s string) bool {
	_, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	return true
}

func isValidDomain(d string) bool {
	_, err := publicsuffix.Parse(d)
	if err != nil {
		return false
	}

	return true
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
