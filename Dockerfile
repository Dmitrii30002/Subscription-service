FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o /app/bin/server ./cmd/main.go

FROM alpine:latest

COPY config.yaml /config.yaml

COPY --from=builder /app/bin/server /app/server

CMD ["/app/server"]