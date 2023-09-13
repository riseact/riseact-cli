
build-linux:
	GOOS=linux GOARCH=amd64 go build -o dist/riseact cmd/main.go

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o dist/riseact cmd/main.go

build-windows:
	GOOS=windows GOARCH=amd64 go build -o dist/riseact.exe cmd/main.go

build: 
	goreleaser release --snapshot --clean

release: 
	goreleaser release

dev:
	go run cmd/main.go auth login

install:
	${MAKE} build-linux
	cp dist/riseact ~/.go/bin

codegen: 
	echo "Generating GraphQL client code..."
	go run github.com/Khan/genqlient 
