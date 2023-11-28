# See https://hub.docker.com/r/shaspitz/geth-poa
# Version: v0

FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev linux-headers git make

RUN git clone https://github.com/shaspitz/go-ethereum.git /go-ethereum
WORKDIR /go-ethereum

# commit: lets try hardcoding delay 
RUN git checkout 9bb1a7f0034002e79c4a91406ea3828ad3e4627f
RUN make geth

FROM alpine:latest

RUN apk add --no-cache jq

COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

COPY genesis.json /genesis.json

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8545

ENTRYPOINT ["/entrypoint.sh"] 
