# Monitor service Dockerfile
# Builds the price monitor that indexes blockchain events and stores prices
FROM golang:1.25-alpine

WORKDIR /app

# Cache Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build binary
COPY . .
RUN go build -o monitor ./cmd/monitor

# Expose metrics/health endpoint
EXPOSE 8082

CMD ["./monitor"]
