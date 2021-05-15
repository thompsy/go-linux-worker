LINT_TARGETS=./cmd/... ./lib/... ./testing/...
UNIT_TESTS=./cmd/... ./lib/...
INTEGRATION_TESTS=./testing/...

.PHONY: all
all: clean certs protoc build lint test

.PHONY: protoc
protoc:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./lib/protobuf/api.proto

.PHONY: build
build: protoc
	go fmt ./...
	go mod vendor
	GOOS=linux go build -o ./bin/server cmd/server/main.go
	go build -o ./bin/client cmd/client/main.go

.PHONY: install-tools
install-tools:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc

.PHONY: install-linter
install-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v1.39.0

.PHONY: docker-build
docker-build:
	docker build -t worker-api-server .

.PHONY: docker-run
docker-run:
	docker run \
		--privileged \
		-p 8080:8080 \
		worker-api-server:latest

.PHONY: lint
lint:
	./bin/golangci-lint run $(LINT_TARGETS)

certs:
	./scripts/generate-mtls-certs.sh

.PHONY: test
test: certs
	go test -v -timeout 30s ./...

.PHONY: integ-test
integ-test: certs
	docker-compose build
	docker-compose up \
		--abort-on-container-exit \
		--exit-code-from test
	docker-compose down

.PHONY: clean
clean:
	rm -rf certs bin lib/protobuf/*.go

