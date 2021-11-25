package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	registrypkg "github.com/tdex-network/tor-proxy/pkg/registry"
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
		&cli.StringFlag{
			Name:  "tls-cert-path",
			Usage: "path of the the TLS certificate",
		},
		&cli.StringFlag{
			Name:  "tls-key-path",
			Usage: "path of the the TLS key",
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

// auto update period 
const autoUpdatePeriod time.Duration = 12 * time.Hour

func startAction(ctx *cli.Context) error {
	// create registry
	registry, err := registrypkg.NewRegistry(ctx.String("registry"))
	if err != nil {
		return fmt.Errorf("laoding json: %w", err)
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

	// Add registry to the proxy
	// this will init the set of redirects
	proxy.WithRegistry(registry)
	stopAutoUpdateFunc := func() {} // will be used to stop the auto update goroutine

	if proxy.Registry.RegistryType() == registrypkg.RemoteRegistryType {
		errorHandler := func (err error) {
			log.Println("registry auto update error: %w", err) 
		}
		// start auto updater and store the stop function for shutdown
		stopAutoUpdateFunc = proxy.StartAutoUpdateRedirects(autoUpdatePeriod, errorHandler)
	}


	// check if insecure flag, otherwise either domain or key & cert paths MUST be present to serve with TLS
	var address string
	var tlsOptions *torproxy.TLSOptions
	if ctx.Bool("insecure") {
		address = ":" + fmt.Sprint(ctx.Int("port"))
	} else {
		email := ctx.String("email")
		domain := ctx.String("domain")
		tlsKey := ctx.String("tls-key-path")
		tlsCert := ctx.String("tls-cert-path")

		if (tlsKey == "" && tlsCert != "") || (tlsKey != "" && tlsCert == "") {
			return fmt.Errorf(
				"TLS requires both key and certificate when enabled",
			)
		}

		// check if given domain is valid URL
		if (len(domain) == 0 || !isValidDomain(domain)) && tlsKey == "" && tlsCert == "" {
			return errors.New("either domain or certificate is required for TLS. Do you want to use --insecure?")
		}

		address = ":443"
		tlsOptions = &torproxy.TLSOptions{
			Domains: []string{domain},
			Email:   email,
			TLSKey:  tlsKey,
			TLSCert: tlsCert,
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
	stopAutoUpdateFunc()

	fmt.Println("Shutdown")

	return nil
}

func isValidDomain(d string) bool {
	_, err := publicsuffix.Parse(d)
	return err == nil
}
