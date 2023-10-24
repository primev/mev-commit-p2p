# RPC APIs

## Overview

There's two key RPC APIs this software provides:
- Searcher API
- Builder Internal Operations API

## Searcher API
- This is the api that takes bids into the mev-commit node that is emulating a searcher. 
- The SendBid RPC endpoint will subseqently propegate the Bid after it is signed, to the primev P2P network.

The format for the request payload is as follows:

```protobuf
message Bid {
  string tx_hash = 1;
  int64 amount = 2;
  int64 block_number = 3;
};
```

Which is the following in JSON Format:
```javascript
{
  "tx_hash": "<string transaction hash>",
  "amount": <integer amount of bid in wei>,
  "block_number": <the block number formated as a base 10 integer>
}
```

The response to the searcher API is a stream of commitments, an example response is shown below:
```javascript
{
    "tx_hash": "transaction_hash15",
    "amount": "1000",
    "block_number": "12345",
    "bid_digest": "fb77987f64d8efaa93c659e4365e60ba7b1b3013ee12b4c988e3dbd87b76109d",
    "bid_signature": "65cb64450be1c83e48a3de5565c07d10b69a75c6c463af01ffb20849e777861a3fd07e1415c83f31f1e05cc7b430b4073faf988b3b0a469148e02ccba9fd6d9901",
    "pre_confirmation_digest": "0f25c2d8adc489d2db535865c70a47ab7eccbbc89ca95b705547c38811712111",
    "pre_confirmation_signature": "4838b53968be8a4cd4bceee9a8299885546b7d184cfe6390dcb8afd37fec3c1b08f0ce03935afce5b11b9f425434a4b22d01cb4d4dd5f4e5894c699302dbb3ad01"
}
```


## Commitments from Builders | Builder API
To gather commitments from builders, the builder mev-commit node must maintain an active service that interfaces with the [GRPC API](https://github.com/primevprotocol/mev-commit/blob/main/rpc/builderapi/v1/builderapi.proto) and interacts with the following functions:

```protobuf
  // ReceiveBids is called by the builder to receive bids from the mev-commit node.
  // The mev-commit node will stream bids to the builder.
  rpc ReceiveBids(EmptyMessage) returns (stream Bid) {}
  // SendProcessedBids is called by the builder to send processed bids to the mev-commit node.
  // The builder will stream processed bids to the mev-commit node.
  rpc SendProcessedBids(stream BidResponse) returns (EmptyMessage) {}

```

**By default this service is disabled**, and must be enabled by setting the BuilderAPIEmabled flag in the config file to true.

The file is located at [./config/builder.yaml](../../config/builder.yml) form the top level of the project and the variable is set to `expose_builder_api: false` by default.

