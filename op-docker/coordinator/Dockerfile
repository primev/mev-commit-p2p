FROM node:20.9-bullseye AS base-node
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
RUN git clone --branch v1.2.0 --depth 1 --recurse-submodules https://github.com/ethereum-optimism/optimism.git .
RUN pnpm install 
RUN pnpm build

# Install ssh client
RUN apt-get install -y openssh-client

# Add GitHub to known hosts for SSH
RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts

ARG SSH_PRIVATE_KEY

# Host must run SSH_PRIVATE_KEY="$(cat ~/.ssh/id_rsa)" or similar
RUN if [ -z "$SSH_PRIVATE_KEY" ]; then echo "SSH_PRIVATE_KEY env var must be set to valid key (often at ~/.ssh)" >&2; exit 1; fi

# Write the SSH private key into a file
RUN echo "$SSH_PRIVATE_KEY" > /root/.ssh/id_rsa && \
    chmod 600 /root/.ssh/id_rsa

# Clone the private repo
WORKDIR /
RUN git clone git@github.com:primevprotocol/rollup-preconf.git
WORKDIR /rollup-preconf

# Checkout latest commit from shawn/downgrade-solidity as of 11/8/2023 
RUN git checkout e20841ee2143d60724e2094d7c847eecc76e97ef && git submodule update --init --recursive
RUN npm install 
RUN npx hardhat compile

# Set env vars for l2 contract deployment, note this account will be funded during coordinator setup
# Address:     0x17b9A666Db52274090236DffD7f7aF589E4d9F8C
# Private key: 0xaa17da7fd48f4b82f6a04efe3f52a065d505fe0b36486cf96eff99637d16c2e1
ENV SIGNER_ADDRESS 0x17b9A666Db52274090236DffD7f7aF589E4d9F8C
ENV SIGNER_PRIVATE_KEY aa17da7fd48f4b82f6a04efe3f52a065d505fe0b36486cf96eff99637d16c2e1     

ENV GOPATH /go
WORKDIR /shared-optimism
RUN make pre-devnet

ENV PATH $PATH:$GOPATH/bin

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
