FROM node:21-bullseye AS base-node
WORKDIR /node

RUN npm install -g pnpm

FROM golang:1.21-bullseye

RUN apt-get update && \
    apt-get install -y git curl jq make bash openssl python3 build-essential ca-certificates

ENV PYTHON python3

RUN curl -L https://foundry.paradigm.xyz | bash
ENV PATH="${PATH}:/root/.foundry/bin"
RUN foundryup

COPY --from=base-node /usr/local /usr/local

WORKDIR /shared-optimism
RUN git clone --recurse-submodules https://github.com/ethereum-optimism/optimism.git .
RUN git checkout v1.2.0
RUN pnpm install 
RUN pnpm build

ENV GOPATH /go
RUN make pre-devnet

ENV PATH $PATH:$GOPATH/bin

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
