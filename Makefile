
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

install:
	${MAKE} build-linux
	cp dist/riseact ~/.go/bin

codegen: 
	echo "Generating GraphQL client code..."
	go get github.com/Khan/genqlient/generate@v0.6.0
	go run github.com/Khan/genqlient 
