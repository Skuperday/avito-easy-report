# Stage 1: сборка
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server/

# Stage 2: минимальный образ
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/server .

EXPOSE 8080

ENV PORT=8080
ENV JWT_SECRET=avito-secret-change-in-production

CMD ["./server"]
