# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s" \
    -o /out/chat-app ./cmd/server



FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H appuser \
    && mkdir -p /app/logs \
    && chown -R appusr:appuser /app

COPY --from=builder /out/chat-app /app/chat-app

USER appuser

EXPOSE 8181

CMD ["/app/chat-app"]