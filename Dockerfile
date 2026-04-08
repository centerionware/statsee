# ---------- Frontend Build ----------
FROM node:20-alpine AS frontend
WORKDIR /frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install

COPY frontend .
RUN npm run build


# ---------- Go Build ----------
FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY . .

# Init module if not present (safe)
RUN go mod init github.com/centerionware/statsee

# Copy go files first (cache deps)
# COPY go.mod go.sum* ./
# RUN go mod tidy


# Copy built frontend into static dir (for embed)
COPY --from=frontend /frontend/dist ./static

# Install ALL required deps (original + new)
RUN go get \
    github.com/gorilla/websocket \
    github.com/shirou/gopsutil/v3/cpu \
    github.com/shirou/gopsutil/v3/disk \
    github.com/shirou/gopsutil/v3/mem \
    github.com/shirou/gopsutil/v3/net \
    go.etcd.io/bbolt \
    github.com/showwin/speedtest-go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app


# ---------- Final ----------
FROM scratch
COPY --from=builder /app/app /app
# COPY --from=golang:1.23-alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/app"]