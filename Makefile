BINARY_NAME=riseact

build-windows:
	GOOS=windows GOARCH=amd64 go build -o releases/$(BINARY_NAME).exe ./cmd/main.go

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o releases/$(BINARY_NAME)_mac ./cmd/main.go

build-linux:
	GOOS=linux GOARCH=amd64 go build -o releases/$(BINARY_NAME) ./cmd/main.go

build-all: build-windows build-darwin build-linux

dev:
	go run cmd/main.go auth login

install:
	${MAKE} build
	mv riseact ~/.go/bin

codegen: 
	echo "Generating GraphQL client code..."
	go run github.com/Khan/genqlient 
