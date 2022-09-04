build-win:
	CGO_ENABLE=0 GOOS=windows go build -o bin/win/transfer.exe

build-darwin:
	CGO_ENABLE=0 GOOS=darwin go build -o bin/darwin/transfer

build-linux:
	CGO_ENABLE=0 GOOS=linux go build -o bin/linux/transfer

build-all: build-darwin build-linux build-win

build-gui:
	CGO_ENABLE=0 GOOS=windows go build -o bin/win/transfer-gui.exe ui/main.go
	GOARCH=wasm GOOS=js go build -o bin/web/app.wasm ui/main.go

test:
	go test ./... -coverprofile coverage.out
