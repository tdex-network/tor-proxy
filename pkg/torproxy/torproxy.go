package torproxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
	"golang.org/x/net/proxy"
)

// TorProxy holds the tor client details and the list cleartext addresses to be redirect to the onions
type TorProxy struct {
	Address   string
	Client    *TorClient
	Redirects []string
}

// TorClient holds the Host and Port for the tor client and if useEmbedded=true means is using an embedded tor client instance
type TorClient struct {
	tor *tor.Tor

	useEmbedded bool

	Host string
	Port int
}

// StartTor starts the embedded tor client
func (tp *TorProxy) StartTor() error {
	// Starting tor please wait a bit...
	torClient, err := tor.Start(nil, &tor.StartConf{
		NoAutoSocksPort: true,
		ProcessCreator:  libtor.Creator,
	})
	if err != nil {
		return fmt.Errorf("Failed to start tor: %v", err)
	}

	tp.Client = &TorClient{
		tor:         torClient,
		useEmbedded: true,
	}

	return nil
}

// Stop stops the tor client
func (tp *TorProxy) Stop() error {
	return tp.Client.tor.Close()
}

// NewTorProxyFromHostAndPort returns a *TorProxy with givnen host and port
func NewTorProxyFromHostAndPort(address string, torHost string, torPort int) *TorProxy {
	return &TorProxy{
		Address: address,
		Client: &TorClient{
			Host: torHost,
			Port: torPort,
		},
	}
}

// NewTorProxy returns a default *TorProxy connecting on canonical localhost:9050
func NewTorProxy(address string) *TorProxy {
	return &TorProxy{
		Address: address,
		Client: &TorClient{
			Host: "127.0.0.1",
			Port: 9050,
		},
	}
}

// WithRedirects modify the TorProxy struct with givend from -> to map
func (tp *TorProxy) WithRedirects(redirects []string) {
	tp.Redirects = append(tp.Redirects, redirects...)
}

// Serve ...
func (tp *TorProxy) Serve() (err error) {

	// Create a socks5 dialer
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", tp.Client.Host, tp.Client.Port), nil, proxy.Direct)
	if err != nil {
		log.Fatalf("Couldn't connect to socks proxy: %s", err.Error())
	}

	if err := reverseProxy(tp.Address, tp.Redirects, dialer); err != nil {
		return err
	}

	return
}

func reverseProxy(address string, redirects []string, dialer proxy.Dialer) error {

	for _, to := range redirects {

		origin, err := url.Parse(to)
		if err != nil {
			return err
		}

		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
			req.Host = origin.Host
		}

		transport := &http.Transport{
			Dial: dialer.Dial,
		}

		revproxy := &httputil.ReverseProxy{Director: director, Transport: transport}

		pattern := "/" + withoutOnion(origin.Host)
		http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			revproxy.ServeHTTP(w, r)
		})

	}

	return http.ListenAndServe(address, nil)
}

func withoutOnion(host string) string {
	hostWithoutPort, _, _ := net.SplitHostPort(host)
	return strings.ReplaceAll(hostWithoutPort, ".onion", "")
}
