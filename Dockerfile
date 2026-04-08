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
RUN apk add git 
RUN go mod tidy

COPY --from=frontend /frontend/dist ./static

RUN CGO_ENABLED=0 GOOS=linux go build -o app


# ---------- Final ----------
FROM scratch
COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]