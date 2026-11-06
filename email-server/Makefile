build:
	go build -trimpath -ldflags="-s -w" -o email-server .
	$(if $(shell command -v upx), upx email-server)

mac:
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o email-server-darwin .

win:
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o email-server.exe .
	$(if $(shell command -v upx), upx email-server.exe)

linux:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o email-server-linux .

test:
	go build ./...
	go vet ./...
	go test ./...

.PHONY: build mac win linux test
