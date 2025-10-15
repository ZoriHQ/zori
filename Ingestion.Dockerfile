FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zori .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/zori .
COPY --from=builder /app/ipdb.mmdb .

RUN chown -R appuser:appgroup /app

USER appuser

ENTRYPOINT ["./zori"]
CMD ["i"]

EXPOSE 1324
