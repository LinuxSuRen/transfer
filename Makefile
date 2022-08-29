build-win:
	CGO_ENABLE=0 GOOS=windows go build -o bin/transfer.exe