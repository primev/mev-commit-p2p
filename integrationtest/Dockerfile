FROM golang:1.21.1 AS builder

WORKDIR /app
COPY . .

ARG node_type

RUN CGO_ENABLED=0 GOOS=linux make build

FROM alpine:latest

ARG node_type
ENV NODE_TYPE=${node_type}

COPY --from=builder /app/bin/mev-commit /app/mev-commit
COPY --from=builder /app/integrationtest/config/${NODE_TYPE}.yaml /config.yaml
COPY --from=builder /app/integrationtest/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 13522 13523 13524

ENTRYPOINT ["/entrypoint.sh"]

