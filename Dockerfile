FROM golang:1.23.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API binary
RUN go build -o bin/api ./cmd/api

# Build goose CLI
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.26.0


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Qyzylorda

WORKDIR /app

# API
COPY --from=builder /app/bin/api /app/api

# goose
COPY --from=builder /go/bin/goose /usr/local/bin/goose

# migrations (from repo root)
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

CMD ["/app/api"]
