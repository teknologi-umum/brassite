FROM golang:1.27.1-alpine3.24 AS builder

WORKDIR /build

COPY . .

RUN go build -o brassite -ldflags="-X main.version=$(git rev-parse HEAD)" ./cmd/brassite/

FROM alpine:3.24 AS runtime

WORKDIR /usr/local/src/brassite

COPY . .

COPY --from=builder /build/brassite /usr/local/bin/brassite

CMD ["/usr/local/bin/brassite"]