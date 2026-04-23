FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rest-service ./cmd/app

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

RUN addgroup -S app && adduser -S app -G app

WORKDIR /root/

COPY --from=builder /app/rest-service .
COPY --from=builder /app/migrations ./migrations

RUN chown -R app:app ./

USER app

COPY .env ./

EXPOSE 8080

CMD ["./rest-service"]