FROM golang:1.24.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/bin/node ./cmd/node
RUN go build -o /app/bin/client ./cmd/client


FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/node /app/node
COPY --from=builder /app/bin/client /app/client

EXPOSE 5001
EXPOSE 8001

CMD ["/app/node"]