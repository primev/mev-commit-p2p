FROM golang:1.21.1 AS builder

WORKDIR /app
COPY . .

RUN cd integrationtest/bidder && \
    CGO_ENABLED=0 GOOS=linux go build -o bidder-emulator main.go

FROM alpine:latest

COPY --from=builder /app/integrationtest/user/user-emulator /app/user-emulator
COPY --from=builder /app/integrationtest/user/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]

