.PHONY: all
all: install-tools lint test

.PHONY: install-tools
install-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v1.39.0

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test: certs
	go test -race -cover -timeout 30s ./...

.PHONY: clean
clean:
	rm -rf bin

