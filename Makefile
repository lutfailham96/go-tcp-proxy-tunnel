.PHONY: format build clean

format:
	find . -name "*.go" -not -path ".git/*" | xargs gofmt -s -d -w

build:
	@echo "Building go-tcp-proxy-tunnel binary"
	@go build -ldflags="-w -s" -o go-tcp-proxy-tunnel github.com/lutfailham96/go-tcp-proxy-tunnel/cmd/tcp-proxy-tunnel
	@go build -ldflags="-w -s" -o go-ws-web-server github.com/lutfailham96/go-tcp-proxy-tunnel/cmd/ws-web-server/main.go
	@echo "Generated proxy executable: ${PWD}/go-tcp-proxy-tunnel"
	@echo "Generated web server executable: ${PWD}/go-ws-web-server"

install:
	@cp -ap ${PWD}/go-tcp-proxy-tunnel /usr/local/bin
	@cp -ap ${PWD}/go-ws-web-server /usr/local/bin

clean:
	@echo "Cleaning unused file"
	rm go-tcp-proxy-tunnel
