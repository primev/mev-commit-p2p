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
RUN git clone --recurse-submodules https://github.com/ethereum-optimism/optimism.git .
RUN git checkout v1.2.0
RUN pnpm install 
RUN pnpm build

# Install ssh client
RUN apt-get install -y openssh-client

# Add GitHub to known hosts for SSH
RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts

ARG SSH_PRIVATE_KEY

# Write the SSH private key into a file
RUN echo "$SSH_PRIVATE_KEY" > /root/.ssh/id_rsa && \
    chmod 600 /root/.ssh/id_rsa

# Clone the private repo
WORKDIR /
RUN git clone git@github.com:primevprotocol/rollup-preconf.git
WORKDIR /rollup-preconf
RUN npm install && npx hardhat compile

# For l2 contract deployment, rm prefix?
ENV PRIVATE_KEY aa17da7fd48f4b82f6a04efe3f52a065d505fe0b36486cf96eff99637d16c2e1     

ENV GOPATH /go
WORKDIR /shared-optimism
RUN make pre-devnet

ENV PATH $PATH:$GOPATH/bin

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
