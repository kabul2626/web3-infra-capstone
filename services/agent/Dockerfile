# Agent service Dockerfile
# Builds the price oracle agent that fetches prices and submits to smart contract
FROM golang:1.25-alpine

WORKDIR /app

# Cache Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build binary
COPY . .
RUN go build -o agent ./cmd/agent

# Expose metrics/health endpoint
EXPOSE 8081

CMD ["./agent"]