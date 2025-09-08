# Builder
FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN go env -w GOPROXY=https://goproxy.io,direct && \
    go env -w GOSUMDB=off && \
    echo "precedence ::ffff:0:0/96  100" >> /etc/gai.conf

COPY ../go.mod ../go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /ltp-service ./cmd/ltp-service

# Final
FROM alpine:latest
COPY --from=builder /ltp-service /ltp-service
COPY ../config/local.yaml /tmp
EXPOSE 8080
ENTRYPOINT ["/ltp-service"]
