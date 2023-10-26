FROM us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node:v1.2.0

RUN apk update && apk add --no-cache curl 

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
