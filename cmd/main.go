package main

import (
	"log"
	"os"

	"github.com/tdex-network/tor-proxy/pkg/torproxy"
)

func main() {

	fromTo := make(map[string]string, 0)
	fromTo["localhost:7070"] = os.Args[1]

	proxy := torproxy.NewTorProxyFromHostAndPort("127.0.0.1", 9150)
	proxy.WithRedirects(fromTo)

	log.Println("Serving tor proxy")
	if err := proxy.Serve(); err != nil {
		log.Panicln(err)
	}

}
