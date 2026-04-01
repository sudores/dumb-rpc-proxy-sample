FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o proxy ./cmd/server

# ---

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/proxy /proxy
COPY config.yaml /config.yaml

ENV CONFIG_FILE=/config.yaml

EXPOSE 8080

ENTRYPOINT ["/proxy"]
