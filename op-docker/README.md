# op-devnet

This directory houses the dockerfiles and entrypoints to spin up a local op-stack rollup as the mev-commit settlement layer. 

Implementation is inspired by https://github.com/ethereum-optimism/optimism/tree/develop/bedrock-devnet, while prioritizing full dockerization.

Key system params can be set from `.env` in root directory of repo.

## Design

*coordinator* is responsible for general setup of both L1 and L2. Genesis state creation, L1 contract deployment (enacted with emphemeral dev-mode geth), and initiazing processes with custom config.

*l1-geth* is a single node geth instance running a POA protocol, Clique. 

*op-geth* is the execution client for the L2 rollup. 

*op-node* is the psuedo consensus client for the L2 rollup, primarily responsible for deriving L2 state from submitted data on L1. 

*op-batcher* takes transactions from the L2 Sequencer and publishes those transactions to L1.

*op-proposer* proposes new state roots for L2.
