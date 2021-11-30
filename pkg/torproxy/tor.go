package torproxy

import (
	"context"
	"fmt"
	"os"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
)

// TorClient holds the Host and Port for the tor client and if useEmbedded=true means is using an embedded tor client instance
type TorClient struct {
	tor *tor.Tor

	useEmbedded bool

	Host string
	Port int
}

// NewTorEmbedded starts the embedded tor client
func NewTorEmbedded() (*TorClient, error) {
	// Starting tor please wait a bit...
	torInstance, err := tor.Start(context.Background(), &tor.StartConf{
		NoAutoSocksPort: true,
		ProcessCreator:  libtor.Creator,
		DebugWriter:     os.Stderr,
		EnableNetwork:   true,
		DataDir:         "/tmp/tordir",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start tor: %v", err)
	}

	return &TorClient{
		Host:        "127.0.0.1",
		Port:        9050,
		tor:         torInstance,
		useEmbedded: true,
	}, nil

}

// Close stops the tor client
func (tc *TorClient) Close() error {
	return tc.tor.Close()
}
