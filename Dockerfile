FROM golang:1.16-alpine AS builder
WORKDIR /go/src/app
RUN apk add protobuf make build-base openssl && \
	go get -u github.com/golang/protobuf/protoc-gen-go && \
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
COPY . .
RUN make certs build

FROM golang:1.16-alpine as server
WORKDIR /go/src/app
COPY --from=builder /go/src/app/bin/server .
COPY --from=builder /go/src/app/certs/ca.crt ./certs/
COPY --from=builder /go/src/app/certs/server.crt ./certs
COPY --from=builder /go/src/app/certs/server.key ./certs
EXPOSE 8080/tcp
CMD ["./server"]
