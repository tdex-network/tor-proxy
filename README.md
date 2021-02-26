# 🧅 tor-proxy
HTTP/1.x reverse proxy to consume onion endpoints without installing a tor client 


## TL;DR

Everything that comes to proxy.tdex.network/<onion_public_key> is redirected to <onion_public_key>.onion

## 📩 Install 

1. [Download the latest release for MacOS or Linux](https://github.com/tdex-network/tor-proxy/releases)

2. Move it into a folder in your PATH (eg. `/usr/local/bin`) and rename as `torproxy`

3. Give executable permissions. (eg. `chmod a+x /usr/local/bin/torproxy`)

## ℹ️ Usage

### `start` command


By default you should have a Tor client running on the canonical `9050` port. You can change that with `--socks5-hostname` and `--socks5-port` or use the embedded tor client with `--use-tor`

* Run *cleartext* on default port :7070

```sh
$ torproxy start --insecure --registry '[{"endpoint": "http://somewherefaraway.onion:80"}]' 
```

* Run *with SSL* 

```sh
$ torproxy start --domain mywebsite.com --registry '[{"endpoint": "http://somewherefaraway.onion:80"}]' 
```

* Load registry from a remote URL 

```sh
$ torproxy start --domain mywebsite.com --registry https://raw.githubusercontent.com/tdex-network/tdex-registry/master/registry.json
```


* Load registry from local path to file

```sh
$ torproxy start --domain mywebsite.com --registry ./registry.json
```

* Use embedded tor client

```sh
$ torproxy start --domain mywebsite.com --registry ./registry.json --use-tor 
```
## 🐋 Docker

* Build

```sh
$ docker build -t ghcr.io/tdex-network/torproxy .
```

* Run 

```sh
$ docker run -it -d -p 443:443 -p 80:80 --name proxy --restart unless-stopped ghcr.io/tdex-network/torproxy start --use-tor --domain proxy.tdex.network --email myemail@domain.com --registry https://raw.githubusercontent.com/tdex-network/tdex-registry/master/registry.json 
```
