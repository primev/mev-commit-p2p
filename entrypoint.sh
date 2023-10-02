#!/bin/sh

echo "Node Type: $NODE_TYPE"

# Define paths
KEY_PATH="/keys"
CONFIG_PATH="/config"

# Generate the private key based on node type
if [ "$NODE_TYPE" = "bootnode" ]; then
    /app/mev-commit create-key ${KEY_PATH}/bootnode
    PRIV_KEY_FILE="${KEY_PATH}/bootnode"
    CONFIG_FILE="${CONFIG_PATH}/bootnode.yml"
elif [ "$NODE_TYPE" = "builder" ]; then
    /app/mev-commit create-key ${KEY_PATH}/builder
    PRIV_KEY_FILE="${KEY_PATH}/builder"
    CONFIG_FILE="${CONFIG_PATH}/builder.yml"
else
    /app/mev-commit create-key ${KEY_PATH}/searcher
    PRIV_KEY_FILE="${KEY_PATH}/searcher"
    CONFIG_FILE="${CONFIG_PATH}/searcher.yml"
fi

# Update the private key path in the configuration
sed -i "s|priv_key_file:.*|priv_key_file: ${PRIV_KEY_FILE}|" ${CONFIG_FILE}

# If this is not the bootnode, update the bootnodes entry with P2P ID
if [ "$NODE_TYPE" != "bootnode" ]; then
    # Wait for a few seconds to ensure the bootnode is up and its API is accessible
    sleep 10

    BOOTNODE_RESPONSE=$(curl -s bootnode:13523/topology)
    BOOTNODE_P2P_ID=$(echo "$BOOTNODE_RESPONSE" | jq -r '.self.Underlay')
    BOOTNODE_IP=$(getent hosts bootnode | awk '{ print $1 }')

    echo "Response from bootnode:"
    echo "$BOOTNODE_RESPONSE"

    if [ -n "$BOOTNODE_P2P_ID" ]; then
        sed -i "s|<p2p_ID>|${BOOTNODE_P2P_ID}|" ${CONFIG_FILE}
	sed -i "s|<localhost>|${BOOTNODE_IP}|" ${CONFIG_FILE}
    else
        echo "Failed to fetch P2P ID from bootnode. Exiting."
        exit 1
    fi
fi

echo "starting mev-commit with config file: ${CONFIG_FILE}"
/app/mev-commit start --config ${CONFIG_FILE}

