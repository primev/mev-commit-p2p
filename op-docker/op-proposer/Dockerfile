FROM us-docker.pkg.dev/oplabs-tools-artifacts/images/op-proposer:v1.2.0

RUN apk update && apk add --no-cache curl jq

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
