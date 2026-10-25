FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o worker-cn-2api .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/worker-cn-2api .
ENV WORKER_HOST=0.0.0.0 \
    WORKER_PORT=10081 \
    WORKER_DATA_PATH=/app/data/data.json \
    TZ=Asia/Shanghai
EXPOSE 10081
CMD ["./worker-cn-2api"]
