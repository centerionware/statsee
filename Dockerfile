FROM golang:1.22 AS builder
WORKDIR /app
COPY . .

RUN go mod init github.com/centerionware/statsee && \
    go get github.com/gorilla/websocket \
           github.com/shirou/gopsutil/v3/... \
           go.etcd.io/bbolt && \
    CGO_ENABLED=0 go build -o app

FROM scratch
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]