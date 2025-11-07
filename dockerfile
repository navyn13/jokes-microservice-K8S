# Build stage
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o joke-server .

# Runtime stage
FROM alpine:3.20
RUN adduser -D app
USER app
WORKDIR /home/app
COPY --from=build /app/joke-server .
EXPOSE 8080
ENV PORT=8080
CMD ["./joke-server"]
