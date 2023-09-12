
build-linux:
	GOOS=linux GOARCH=amd64 go build -o dist/riseact cmd/main.go

build: 
	goreleaser release --snapshot --clean

release: 
	goreleaser release --snapshot --clean

dev:
	go run cmd/main.go auth login

build-linux:
	${MAKE} build
	cp dist/riseact ~/.go/bin

codegen: 
	echo "Generating GraphQL client code..."
	go run github.com/Khan/genqlient 
