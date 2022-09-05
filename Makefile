build-win:
	CGO_ENABLE=0 GOOS=windows go build -o bin/win/transfer.exe

build-darwin:
	CGO_ENABLE=0 GOOS=darwin go build -o bin/darwin/transfer

build-linux:
	CGO_ENABLE=0 GOOS=linux go build -o bin/linux/transfer

build-all: build-darwin build-linux build-win

build-gui:
	CGO_ENABLE=0 GOOS=windows go build -ldflags -H=windowsgui -o bin/win/transfer-gui.exe ui/main.go
	CGO_ENABLE=0 GOOS=windows go build -o bin/win/transfer-gui.exe ui/main.go

test:
	go test ./... -coverprofile coverage.out
