package torproxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/proxy"
)

// TorProxy holds the tor client details and the list cleartext addresses to be redirect to the onions URLs
type TorProxy struct {
	Address   string
	Client    *TorClient
	Redirects []*url.URL
}

// NewTorProxyFromHostAndPort returns a *TorProxy with givnen host and port
func NewTorProxyFromHostAndPort(address string, torHost string, torPort int) (*TorProxy, error) {

	// TODO parse tor host and port here and try to dial/check it works

	return &TorProxy{
		Address: address,
		Client: &TorClient{
			Host: torHost,
			Port: torPort,
		},
	}, nil
}

// NewTorProxy returns a default *TorProxy connecting on canonical localhost:9050
func NewTorProxy(address string) (*TorProxy, error) {
	torClient, err := NewTorEmbedded()
	if err != nil {
		return nil, fmt.Errorf("Couldn't start tor client: %w", err)
	}

	return &TorProxy{
		Address: address,
		Client:  torClient,
	}, nil
}

// WithRedirects modify the TorProxy struct with givend from -> to map
func (tp *TorProxy) WithRedirects(redirects []string) error {
	var err error
	for _, to := range redirects {
		// we parse the destination upstram which should be on *.onion address
		origin, err := url.Parse(to)
		if err != nil {
			err = fmt.Errorf("Failed to parse address : %v", err)
			break
		}
		tp.Redirects = append(tp.Redirects, origin)
	}

	return err
}

// Serve starts a HTTP1 Listener aand reverse proxy all the cleartext requests to registered Onion addresses.
// For each onion address we get to know thanks the WithRedirects method, we register a URL.path like
// host:port/<just_onion_host_without_dot_onion>/[<grpc_package>.<grpc_service>/<grpc_method>]
// Each incoming request will be proxied to <just_onion_host_without_dot_onion>.onion/[<grpc_package>.<grpc_service>/<grpc_method>]
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

// reverseProxy takes an address where to listen, a dialer with SOCKS5 proxy and a list of redirects as a list of URLs
// the incoming request should match the pattern host:port/<just_onion_host_without_dot_onion>/<grpc_package>.<grpc_service>/<grpc_method>
func reverseProxy(address string, redirects []*url.URL, dialer proxy.Dialer) error {

	for _, to := range redirects {
		removeForUpstream := "/" + withoutOnion(to.Host)

		// get a simple reverse proxy
		revproxy := generateReverseProxy(to, dialer)

		http.HandleFunc(removeForUpstream+"/", func(w http.ResponseWriter, r *http.Request) {

			// prepare request removing useless headers
			if err := prepareRequest(r); err != nil {
				http.Error(w, fmt.Errorf("preparation request in reverse proxy: %w", err).Error(), http.StatusInternalServerError)
				return
			}

			// remove the <just_onion_host_without_dot_onion> from the upstream path
			pathWithOnion := r.URL.Path
			pathWithoutOnion := strings.ReplaceAll(pathWithOnion, removeForUpstream, "")
			r.URL.Path = pathWithoutOnion

			revproxy.ServeHTTP(w, r)
		})
	}

	return http.ListenAndServe(address, nil)
}

func withoutOnion(host string) string {
	hostWithoutPort, _, _ := net.SplitHostPort(host)
	return strings.ReplaceAll(hostWithoutPort, ".onion", "")
}
