FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o joke-server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/joke-server .
CMD ["./joke-server"]
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o joke-server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/joke-server .
CMD ["./joke-server"]
