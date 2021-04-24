FROM golang:1.16-alpine AS builder
WORKDIR /go/src/app
RUN apk add protobuf make build-base openssl curl
COPY . .
RUN make install-tools install-linter certs build lint test

FROM golang:1.16-alpine as server
WORKDIR /go/src/app
COPY --from=builder /go/src/app/bin/server .
COPY --from=builder /go/src/app/certs/ca.crt ./certs/
COPY --from=builder /go/src/app/certs/server.crt ./certs
COPY --from=builder /go/src/app/certs/server.key ./certs
COPY assets/alpine-minirootfs-3.13.2-x86_64.tar.gz /tmp/alpine.tar.gz
EXPOSE 8080/tcp
CMD ["./server"]
