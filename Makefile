.PHONY: build build-gui test test-race fmt vet check run tidy clean release release-checksums

VERSION ?= dev
LDFLAGS := -s -w -X github.com/wnddd839/codebuddy-proxy/internal/version.Version=$(VERSION)

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/codebuddy-proxy ./cmd/codebuddy-proxy

# GUI 托盘版必须用 GUI 子系统（-H=windowsgui），否则会常驻一个 CMD 黑窗口。
build-gui:
	go build -trimpath -ldflags="$(LDFLAGS) -H=windowsgui" -o bin/codebuddy-proxy-gui.exe ./cmd/codebuddy-proxy-gui

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './bin/*')

vet:
	go vet ./...

check: fmt vet test-race

run:
	go run ./cmd/codebuddy-proxy

tidy:
	go mod tidy

clean:
	rm -rf bin releases

release:
	@mkdir -p releases
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS) -H=windowsgui" -o releases/codebuddy-proxy-gui-windows-x64-$(VERSION).exe ./cmd/codebuddy-proxy-gui
	cp .env.example releases/.env.example
	$(MAKE) release-checksums

release-checksums:
	@cd releases && (sha256sum codebuddy-proxy-* 2>/dev/null || shasum -a 256 codebuddy-proxy-*) > SHA256SUMS.txt
