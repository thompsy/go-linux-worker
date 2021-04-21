.PHONY: all
all: clean protoc build lint test

.PHONY: protoc
protoc:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./lib/protobuf/api.proto

.PHONY: build
build: protoc
	go fmt ./...
	go build ./...

.PHONY: install-tools
install-tools:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v1.39.0

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:
	go test -race -timeout 30s ./...

.PHONY: clean
clean:
	rm -rf bin lib/protobuf/*.go

