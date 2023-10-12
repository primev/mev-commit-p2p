# mev-commit
mev-commit is a P2P software that creates a network of builders and searchers. Searchers can use it to broadcast bids to multiple builders and get pre-confirmations from them.

![](node-architecture.png)

## Quickstart
- The software needs an ECDSA private key. This key creates the ethereum address for the node as well as used for the P2P network. Users can use an existing key or create a new key using the `create-key` command.
```
NAME:
   mev-commit create-key - Create a new ECDSA private key and save it to a file

USAGE:
   mev-commit create-key [command options] <output_file>

OPTIONS:
   --help, -h  show help
```

- Once the key is available the users need to create a yaml config file. Example config files are available in the `config` folder. The important options are defined below:
```
# Path to private key file.
priv_key_file: ~/.mev-commit/keys/nodekey
# Type of peer. Options are builder and searcher.
peer_type: builder
# Port used for P2P traffic. If not configured, 13522 is the default.
p2p_port: 13522
# Port used for HTTP traffic. If not configured, 13523 is the default.
http_port: 13523
# Secret for the node. This is used to authorize the nodes. The value doesnt matter as long as it is sufficiently unique. It is signed using the private key.
secret: hello
# Format used for the logs. Options are "text" or "json".
log_fmt: text
# Log level. Options are "debug", "info", "warn" or "error".
log_level: debug
# Bootnodes used for bootstrapping the network.
bootnodes:
  - /ip4/35.91.118.20/tcp/13524/p2p/16Uiu2HAmAG5z3E8p7o19tEcLdGvYrJYdD1NabRDc6jmizDva5BL3
```

- After the config file is ready, run `mev-commit start` with the config option.
```
NAME:
   mev-commit start - Start the mev-commit node

USAGE:
   mev-commit start [command options] [arguments...]

OPTIONS:
   --config value  path to config file [$MEV_COMMIT_CONFIG]
   --help, -h      show help
```

- After the node is started, users can check the status of the peers connected to the node using the `/topology` endpoint on the HTTP port.
```
{
   self: {
      Addresses: [
         "/ip4/127.0.0.1/tcp/13526",
         "/ip4/192.168.1.103/tcp/13526",
         "/ip4/192.168.100.5/tcp/18625"
      ],
      Ethereum Address: "0x55B3B672DEB14178615F648911e76b7FE1B23e5D",
      Peer Type: "builder",
      Underlay: "16Uiu2HAmBykfyf9A5DnRguHNS1mvSaprzYEkjRf6uafLU4javG4L"
   },
   connected_peers: {
      builders: [
         {
            "0xca61596ccef983eb7cae42340ec553dd89881403"
         }
      ]
   }
}
```

## PreConfirmations from Builders
To recieve preConfirmations from builders, the builder-mev-node needs to have a running service that connects to the RPC endpoints and connects to the following functions:
```protobuf
  // ReceiveBids is called by the builder to receive bids from the mev-commit node.
  // The mev-commit node will stream bids to the builder.
  rpc ReceiveBids(EmptyMessage) returns (stream Bid) {}
  // SendProcessedBids is called by the builder to send processed bids to the mev-commit node.
  // The builder will stream processed bids to the mev-commit node.
  rpc SendProcessedBids(stream BidResponse) returns (EmptyMessage) {}

```

## Sending bids as a Searcher
To send bids, you can use an gRPC api that is availible to searcher nodes. 
Upon running this service, searcher nodes will have access to the following:
```protobuf
service Searcher {
  rpc SendBid(Bid) returns (stream Commitment) {}
}

message Bid {
  string txn_hash = 1;
  int64 bid_amt = 2;
  int64 block_number = 3;
};

```

By default, the docker setup exposes port `13524`, which is the standard port on which the searcher api is running. By hitting SendBid with the bid structure in the following format:
```json
{
  "txn_hash": "<txn-hash>",
  "bid_amt": <number>,
  "block_number": <block-number>
}
```


## Commitments from Builders
To recieve commitments from builders, the builder-mev-node needs to have a running service that connects to the RPC endpoints and connects to the following functions:

## Building Docker Image

To simplify the deployment process, you may utilize Docker to create an isolated environment to run mev-commit.

- Build the Docker Image:
  Navigate to the project root directory (where your Dockerfile is located) and run:
  
  ```
  docker build -t mev-commit:latest .
  ```
- Running with Docker Compose:
 
  ```
    docker-compose up --build
  ```

- Stopping the Service:

  ```
    docker-compose down
  ```