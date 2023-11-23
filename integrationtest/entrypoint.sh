#!/bin/sh

echo "Node Type: $NODE_TYPE"

# If this is not the bootnode, update the bootnodes entry with P2P ID
if [ "$NODE_TYPE" != "bootnode" ]; then
    # Wait for a few seconds to ensure the bootnode is up and its API is accessible
    sleep 10
fi

CONFIG=$(cat /config.yaml)

echo "starting mev-commit with config: ${CONFIG}"
/app/mev-commit start --config /config.yaml
