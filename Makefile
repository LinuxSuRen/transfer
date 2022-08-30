build-win:
	CGO_ENABLE=0 GOOS=windows go build -o bin/transfer.exe

build-darwin:
	CGO_ENABLE=0 GOOS=darwin go build -o bin/transfer

build-linux:
	CGO_ENABLE=0 GOOS=linux go build -o bin/transfer

test:
	go test ./...
