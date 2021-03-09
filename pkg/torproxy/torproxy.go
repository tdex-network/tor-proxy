package torproxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/caddyserver/certmagic"
	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

// TorProxy holds the tor client details and the list cleartext addresses to be redirect to the onions URLs
type TorProxy struct {
	Address   string
	Domains   []string
	Client    *TorClient
	Redirects []*url.URL

	Listener net.Listener
	useTLS   bool
}

// NewTorProxyFromHostAndPort returns a *TorProxy with givnen host and port
func NewTorProxyFromHostAndPort(torHost string, torPort int) (*TorProxy, error) {

	// dial to check if socks5 proxy is listening
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", torHost, torPort), nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("couldn't connect to socks proxy: %w", err)
	}

	tr := &http.Transport{Dial: dialer.Dial}
	c := &http.Client{
		Transport: tr,
	}

	req, err := http.NewRequest(http.MethodGet, "https://check.torproject.org", nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't create request : %w", err)
	}

	_, err = c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't make request: %w", err)
	}

	return &TorProxy{
		Client: &TorClient{
			Host: torHost,
			Port: torPort,
		},
	}, nil
}

// NewTorProxy starts an embedded tor client and returns a default *TorProxy on the canonical localhost:9050
func NewTorProxy() (*TorProxy, error) {
	torClient, err := NewTorEmbedded()
	if err != nil {
		return nil, fmt.Errorf("Couldn't start tor client: %w", err)
	}

	return &TorProxy{
		Client: torClient,
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

// TLSOptions defines the domains we need to obtain and renew a TLS cerficate
type TLSOptions struct {
	Domains    []string
	Email      string
	UseStaging bool
}

// Serve starts a HTTP/1.x reverse proxy for all cleartext requests to the registered Onion addresses.
// An address to listent for TCP packets must be given.
// TLS will be enabled if a non-nil *TLSOptions is given. CertMagic will obtain, store and renew certificates for the domains.
// By default, CertMagic stores assets on the local file system in $HOME/.local/share/certmagic (and honors $XDG_DATA_HOME if set).
// CertMagic will create the directory if it does not exist.
// If writes are denied, things will not be happy, so make sure CertMagic can write to it!
// For each onion address we get to know thanks the WithRedirects method, we register a URL.path like
// host:port/<just_onion_host_without_dot_onion>/[<grpc_package>.<grpc_service>/<grpc_method>]
// Each incoming request will be proxied to <just_onion_host_without_dot_onion>.onion/[<grpc_package>.<grpc_service>/<grpc_method>]
func (tp *TorProxy) Serve(address string, options *TLSOptions) error {

	// Create a socks5 dialer
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", tp.Client.Host, tp.Client.Port), nil, proxy.Direct)
	if err != nil {
		log.Fatalf("couldn't connect to socks proxy: %s", err.Error())
	}

	if options != nil {
		// read and agree to your CA's legal documents
		certmagic.DefaultACME.Agreed = true

		// provide an email address
		if len(options.Email) > 0 {
			certmagic.DefaultACME.Email = options.Email
		}
		// use the staging endpoint while we're developing
		if options.UseStaging {
			certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
		}

		// config
		tlsConfig, err := certmagic.TLS(options.Domains)
		if err != nil {
			return err
		}
		tlsConfig.NextProtos = []string{"http/1.1", http2.NextProtoTLS, "h2-14"} // h2-14 is just for compatibility. will be eventually removed.

		// get a TLS listener
		lis, err := tls.Listen("tcp", address, tlsConfig)
		if err != nil {
			return err
		}

		// Set address and listener
		tp.Address = address
		tp.Listener = lis
		// Set with TLS stuff
		tp.Domains = options.Domains
		tp.useTLS = true
	} else {

		lis, err := net.Listen("tcp", address)
		if err != nil {
			return err
		}

		// Set address and listener
		tp.Address = address
		tp.Listener = lis
	}

	// Now we can reverse proxy all the redirects
	if err := reverseProxy(tp.Redirects, tp.Listener, dialer); err != nil {
		return err
	}

	return err
}

// reverseProxy takes an address where to listen, a dialer with SOCKS5 proxy and a list of redirects as a list of URLs
// the incoming request should match the pattern host:port/<just_onion_host_without_dot_onion>/<grpc_package>.<grpc_service>/<grpc_method>
func reverseProxy(redirects []*url.URL, lis net.Listener, dialer proxy.Dialer) error {

	for _, to := range redirects {
		removeForUpstream := "/" + withoutOnion(to.Host)

		// get a simple reverse proxy
		revproxy := generateReverseProxy(to, dialer)

		http.HandleFunc(removeForUpstream+"/", func(w http.ResponseWriter, r *http.Request) {

			// add cors headers
			addCorsHeader(w, r)

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

	return http.Serve(lis, nil)
}

func withoutOnion(host string) string {
	hostWithoutPort, _, _ := net.SplitHostPort(host)
	return strings.ReplaceAll(hostWithoutPort, ".onion", "")
}

func addCorsHeader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodOptions {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.WriteHeader(http.StatusNoContent)
	return
}
