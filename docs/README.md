# mev-commit

## Summary
Introducing mev-commit, a peer-to-peer (P2P) networking software that serves as a conduit for real-time communication with execution providers. mev-commit enables MEV actors to join and utilize a P2P network for the exchange of execution bids and commitments, enriching the transaction execution experience by allowing for granular specification of execution needs and receiving real-time commitments.

## Actors
The roles of actors within this p2p ecosystem are defined with respect to other actors. A list of possible actors is given below. A good way to interpret them is to observe a given actors' relative placement in the diagram shown below. For example, an MEV Searcher (Transaction Originator) is a user to a Sequencer, and however, that same sequencer can be a user to a block builder. Thus, it's best to think of the roles of actors from the perspective of the MEV Supply Chain. To the left of the diagram are users who source MEV; and to the right of the diagram, are asset holders that help users actualize their MEV. These asset holders are providers in the p2p ecosystem. 

Traditionally, information only moved to the right in this supply chain, from users to providers. With our P2P network, we're allowing information about how the MEV is actualized to flow from providers back to users, along with cryptographic commitments that strengthen the value of information exchange.

![](mev-supply-chain.png)

**Providers**

Providers of execution services (**Block builders, Rollup Sequencers)**

**Users**

Users of execution services (**MEV searchers, AA bundlers, L2s, and other blockspace consumers)**

## Network Topology

The network topology we will be releasing is as follows:

<img src="topology.png" alt="Topology" width="500" height="500"/>

Users will connect to providers, each of these nodes will have access to a bootnode for network startup. Providers will also be able to run gateway nodes to allow users to send bids directly to an RPC endpoint under a provider URL.

## Bids and Privacy

mev-commit is inherently pseudonymous, allowing any Ethereum address to submit a bid for transaction execution, including bids for transactions that belong to others. Bids use the transaction hash identifier for universal provider pickup and are visible to network actors. Bids are processed by both network providers and Primev chain validators, ensuring verifiable commitments and seamless reward settlements.

## Commitments and Privacy

Commitments are commitment signatures from providers in response to bids. mev-commit provides a standard commitment method and a private commitment method for providers to choose from. Private commitments are encrypted and can only be read by the bidder until after the block slot ends and they’re revealed. Providers can also maintain their pseudonymity with commitments, using alternate addresses to obfuscate their identity as known block provider or sequencers.

For more on commitment privacy

## Settlement Layer

Bids and commitments will settle on a specialized Ethereum fork based on the OP stack. Initially centralized, the settlement layer operates as a high-throughput chain to expedite the settlement process. As governance processes are initiated, this chain will become a federated rollup to providers on the network to assume the Sequencer role in turns. A rollup sequencer maintains the state of bids and commitments, acting as a network peer and handles fund settlements, rewards, or slashes.

## Network Flows

Diagram depicting **the flow of bids, commitments, and funds**

<img src="flow.png" alt="Topology" width="750" height="650"/>
