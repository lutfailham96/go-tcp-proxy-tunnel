# go-tcp-proxy-tunnel
[![Maintainability](https://api.codeclimate.com/v1/badges/022a8af7f8393716958d/maintainability)](https://codeclimate.com/github/lutfailham96/go-tcp-proxy-tunnel/maintainability)

Fast & clean tcp proxy tunnel written in Go

This project was intended for debugging text-based protocols. The next version will address binary protocols.

## Install

**Source**

``` sh
$ go get -v github.com/lutfailham96/go-tcp-proxy-tunnel
```

## Usage

```
$ go-tcp-proxy-tunnel --help
Usage of tcp-proxy:
  -l: "127.0.0.1:8082": local address
  -r: "localhost:80": remote address
  -s: "server address / sni address": server:443
  -rp: "use as reverse proxy"
  -op: "local TCP payload replacer""
  -ip: "remote TCP payload replacer""
```

### Client Example

Use custom payload
```shell
$ go run cmd/tcp-proxy/main.go \
    -l 127.0.0.1:9999 \
    -r 127.0.0.1:10443 \
    -s myserver:443 \
    -op "GET ws://cloudflare.com HTTP/1.1[crlf]Host: [host][crlf]Upgrade: websocket[crlf]Connection: keep-alive[crlf][crlf]"

Proxying from 127.0.0.1:9999 to 104.15.50.1:443
```

stunnel configuration
```
[ws]
client = yes
accept = 127.0.0.1:10443
connect = 104.15.50.5:443
sni = cloudflare.com
cert = /etc/stunnel/ssl/stunnel.pem

```

Tunnel over SSH conneciton
```shell
$ ssh -o "ProxyCommand=corkscrew 127.0.0.1 9999 %h %p" -v4ND 1080 my-user@localhost
```

### Todo

* Add unit test
