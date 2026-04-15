FROM mirror.gcr.io/library/golang:1.25-alpine AS builder
RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/server .
COPY ./sql/migrations ./migrations

EXPOSE 8000

CMD ["./server"]