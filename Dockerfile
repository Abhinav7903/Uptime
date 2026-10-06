FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod ./
# If go.sum exists, copy it too
COPY go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build applications
RUN go build -o /api cmd/api/main.go
RUN go build -o /worker cmd/worker/main.go

# Run stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /api .
COPY --from=builder /worker .

# Command will be overridden by docker-compose
CMD ["./api"]
