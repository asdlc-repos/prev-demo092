FROM golang:1.21-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/leave-service .

FROM alpine:3.19

RUN addgroup -S app && adduser -S app -G app
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/leave-service /app/leave-service

USER app
EXPOSE 9090

ENTRYPOINT ["/app/leave-service"]
