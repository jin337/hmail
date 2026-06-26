build:
	go build -ldflags="-s -w" main.go
	$(if $(shell command -v upx), upx email-server)
mac:
	GOOS=darwin go build -ldflags="-s -w" -o email-server-darwin main.go
	$(if $(shell command -v upx), upx email-server-darwin)
win:
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ GOOS=windows GOARCH=amd64 go build -x -v -ldflags "-s -w" -o email-server.exe main.go
	$(if $(shell command -v upx), upx email-server.exe)
linux:
	GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o email-server-linux main.go
	$(if $(shell command -v upx), upx email-server-linux)