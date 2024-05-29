FROM golang:1.22-alpine3.20 AS builder

WORKDIR /build

COPY . .

RUN go build -o brassite ./cmd/brassite/main.go

FROM alpine:3.20 AS runtime

COPY --from=builder /build/brassite /usr/local/bin/brassite

CMD ["/usr/local/bin/brassite"]