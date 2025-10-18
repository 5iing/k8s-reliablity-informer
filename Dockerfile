FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k3s-health-checker .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/k3s-health-checker .
COPY pkg/config/config.yaml /app/config.yaml

CMD ["./k3s-health-checker", "-config", "/app/config.yaml"]
