FROM golang:1.21.1 AS builder

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o user-emulator main.go

FROM alpine:latest

COPY --from=builder /app/user-emulator /app/user-emulator
COPY --from=builder /app/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]

