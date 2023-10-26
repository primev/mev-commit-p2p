FROM us-docker.pkg.dev/oplabs-tools-artifacts/images/op-geth:v1.101301.1

RUN apk add --no-cache jq

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
