FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o file-server .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/file-server .
RUN mkdir -p /app/uploads

EXPOSE 8080
CMD ["./file-server"]
