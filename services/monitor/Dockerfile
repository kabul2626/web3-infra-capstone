FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o monitor ./cmd/monitor

EXPOSE 8082

CMD ["./monitor"]
