# go-tcp-proxy-tunnel
[![Maintainability](https://api.codeclimate.com/v1/badges/022a8af7f8393716958d/maintainability)](https://codeclimate.com/github/lutfailham96/go-tcp-proxy-tunnel/maintainability)

Fast & clean tcp proxy tunnel written in Go

This project was intended for debugging text-based protocols. The next version will address binary protocols.

## Install

**Source**

``` sh
$ go get -v github.com/lutfailham96/go-tcp-proxy-tunnel
```

**Build & install to system**
```sh
$ git clone https://github.com/lutfailham96/go-tcp-proxy-tunnel \
    && cd go-tcp-proxy-tunnel \
    && make build \
    && sudo make install
```

## Usage

```
$ go-tcp-proxy-tunnel --help
Usage of tcp-proxy:
  -l: "127.0.0.1:8082": local address
  -r: "localhost:80": remote address
  -s: "server address / sni address": server:443
  -rp: "use as reverse proxy"
  -op: "local TCP payload replacer"
  -ip: "remote TCP payload replacer"
```

### Server example

Accept incoming connection to use as SSH tunnel
```shell
$ go-tcp-proxy-tunnel \
    -l 127.0.0.1:8082 \
    -r 127.0.0.1:22 \
    -rp

Mode		: reverse proxy
Buffer size	: 65535

go-tcp-proxy-tunnel proxing from 127.0.0.1:8082 to 127.0.0.1:22
```

Nginx site configuration
```
server {
    listen 80;
    listen [::]:80;
    listen 443 ssl;
    listen [::]:443 ssl;

    # SSL configuration
    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;

    server_name my-server;

    location / {
        proxy_pass http://127.0.0.1:8082;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        include proxy_params;
    }
}
```

### Client Example

Use custom payload
```shell
$ go-tcp-proxy-tunnel \
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

### Config File Example

**sever**
```json
{
  "BufferSize": 65535,
  "ReverseProxyMode": true,
  "ProxyInfo": "reverse proxy",
  "LocalAddress": "127.0.0.1:8082",
  "RemoteAddress": "127.0.0.1:22"
}
```

**client**
```json
{
  "BufferSize": 65535,
  "ReverseProxyMode": false,
  "ProxyInfo": "client proxy",
  "LocalAddress": "127.0.0.1:9999",
  "RemoteAddress": "127.0.0.1:10443",
  "LocalPayload": "GET ws://cloudflare.com HTTP/1.1[crlf]Host: [host][crlf]Connection: keep-alive[crlf]Upgrade: websocket[crlf][crlf]",
  "RemotePayload": "HTTP/1.1 200 Connection Established[crlf][crlf]",
  "ServerHost": "my-server:443"
}
```

### Todo

* Add unit test
