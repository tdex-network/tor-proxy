package torproxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
	"golang.org/x/net/proxy"
)

// TorProxy holds the tor client details and the list cleartext addresses to be redirect to the onions
type TorProxy struct {
	Client   *TorClient
	Redirect map[string]string
}

// TorClient holds the Host and Port for the tor client and if UseEmbedded=true will use an embedded tor client instance
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
func NewTorProxyFromHostAndPort(host string, port int) *TorProxy {
	return &TorProxy{
		Client: &TorClient{
			Host: host,
			Port: port,
		},
	}
}

// NewTorProxy returns a default *TorProxy
func NewTorProxy() *TorProxy {
	return &TorProxy{
		Client: &TorClient{
			Host: "127.0.0.1",
			Port: 9050,
		},
	}
}

// WithRedirects modify the TorProxy struct with givend from -> to map
func (tp *TorProxy) WithRedirects(fromTo map[string]string) {
	tp.Redirect = fromTo
}

// Serve ...
func (tp *TorProxy) Serve() (err error) {

	// Create a socks5 dialer
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", tp.Client.Host, tp.Client.Port), nil, proxy.Direct)
	if err != nil {
		log.Fatalf("Couldn't connect to socks proxy: %s", err.Error())
	}

	for from, to := range tp.Redirect {
		if errProxy := reverseProxy(dialer, from, to); errProxy != nil {
			err = errProxy
			break
		}
	}

	return
}

func reverseProxy(dialer proxy.Dialer, from string, to string) error {
	origin, _ := url.Parse(to)

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

	proxy := &httputil.ReverseProxy{Director: director, Transport: transport}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	return http.ListenAndServe(from, nil)
}
