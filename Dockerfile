# Stage 1: Build Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY backend/go.mod ./
COPY backend/*.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o file-server .

# Stage 2: All-in-one image (Nginx + Go backend)
FROM nginx:alpine

RUN apk --no-cache add supervisor ca-certificates

WORKDIR /app
COPY --from=builder /build/file-server .
RUN mkdir -p /app/uploads

COPY frontend/ /usr/share/nginx/html/
COPY nginx/nginx-standalone.conf /etc/nginx/conf.d/default.conf
COPY supervisord.conf /etc/supervisord.conf

EXPOSE 80

CMD ["supervisord", "-n", "-c", "/etc/supervisord.conf"]
