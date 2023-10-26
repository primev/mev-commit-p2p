#!/bin/bash

# Func to be used later in script
wait_on_rpc_server() {
    local URL="$1"
    local COUNTER=0
    local RETRIES=10
    
    while [[ $COUNTER -lt $RETRIES ]]; do
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Trying to connect to RPC server at ${URL}"
        
        if curl -s -X POST "${URL}" -H "Content-type: application/json" \
        -d '{"id":1, "jsonrpc":"2.0", "method": "eth_chainId", "params":[]}' | grep -q "jsonrpc"; then
            echo "$(date '+%Y-%m-%d %H:%M:%S') - RPC server at ${URL} ready"
            break
        fi
        
        sleep 1 # sec 
        COUNTER=$((COUNTER + 1))
    done
    
    if [[ $COUNTER -eq $RETRIES ]]; then
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Timed out waiting for RPC server at ${URL}."
        exit 1
    fi
}

MONOREPO_DIR=/shared-optimism 
DEVNET_DIR="$MONOREPO_DIR/.devnet"
CONTRACTS_BEDROCK_DIR="$MONOREPO_DIR/packages/contracts-bedrock"
DEPLOYMENT_DIR="$CONTRACTS_BEDROCK_DIR/deployments/$DEPLOYMENT_CONTEXT"
OP_NODE_DIR="$MONOREPO_DIR/op-node"
DEPLOY_CONFIG_DIR="$CONTRACTS_BEDROCK_DIR/deploy-config"
DEVNET_CONFIG_PATH="$DEPLOY_CONFIG_DIR/$DEPLOYMENT_CONTEXT.json"
DEVNET_CONFIG_TEMPLATE_PATH="$DEPLOY_CONFIG_DIR/devnetL1-template.json"

TMP_L1_DEPLOYMENTS_PATH="$DEPLOYMENT_DIR/.deploy"
GENESIS_L1_PATH="$DEVNET_DIR/genesis-l1.json"
GENESIS_L2_PATH="$DEVNET_DIR/genesis-l2.json"
ALLOCS_PATH="$DEVNET_DIR/allocs-l1.json"
L1_DEPLOYMENTS_PATH="$DEVNET_DIR/addresses.json" 
ROLLUP_CONFIG_PATH="$DEVNET_DIR/rollup.json"

EPHEMERAL_GETH_URL='http://localhost:8545' 

mkdir -p "$DEVNET_DIR"

echo Starting devnet setup...

if [ ! -e "$GENESIS_L1_PATH" ]; then
    echo "Generating genesis-l1.json"

    if [ ! -e "$ALLOCS_PATH" ]; then
        echo "Generating allocs-l1.json"

        # Use template as base path, 
        # note system can be fragile to mutating certain fields. 
        cat "$DEVNET_CONFIG_TEMPLATE_PATH" > "$DEVNET_CONFIG_PATH"

        # Mutate fields in-place if select env vars are set
        if [ -n "$L1_BLOCK_TIME" ]; then
            jq --arg bt "$L1_BLOCK_TIME" '.l1BlockTime = ($bt | tonumber)' "$DEVNET_CONFIG_PATH" > temp.json && mv temp.json "$DEVNET_CONFIG_PATH"
        fi
        if [ -n "$L2_BLOCK_TIME" ]; then
            jq --arg bt "$L2_BLOCK_TIME" '.l2BlockTime = ($bt | tonumber)' "$DEVNET_CONFIG_PATH" > temp.json && mv temp.json "$DEVNET_CONFIG_PATH"
        fi
        # TODO: figure out where these two fields get reset by end of script
        if [ -n "$GOVERNANCE_TOKEN_SYMBOL" ]; then
            jq --arg gts "$GOVERNANCE_TOKEN_SYMBOL" '.governanceTokenSymbol = $gts' "$DEVNET_CONFIG_PATH" > temp.json && mv temp.json "$DEVNET_CONFIG_PATH"
        fi
        if [ -n "$GOVERNANCE_TOKEN_NAME" ]; then
            jq --arg gtn "$GOVERNANCE_TOKEN_NAME" '.governanceTokenName = $gtn' "$DEVNET_CONFIG_PATH" > temp.json && mv temp.json "$DEVNET_CONFIG_PATH"
        fi

        # config after mutations
        echo "config after mutations: " && cat $DEVNET_CONFIG_PATH

        # Check config for breaking changes
        cd $MONOREPO_DIR
        go run op-chain-ops/cmd/check-deploy-config/main.go --path "$DEVNET_CONFIG_PATH"

        # Spawn ephemeral geth in dev mode to deploy L1 contracts into state
        geth --dev --http --http.api eth,debug \
            --verbosity 4 --gcmode archive --dev.gaslimit 30000000 \
            --rpc.allow-unprotected-txs & # Note & denoting background process

        # Capture PID of process we just started 
        GETH_PID=$!

        # Wait for ephemeral geth to start up
        wait_on_rpc_server "$EPHEMERAL_GETH_URL"

        # Fetch eth_accounts
        DATA=$(curl -s -X POST "${EPHEMERAL_GETH_URL}" -H "Content-type: application/json" \
            -d '{"id":2, "jsonrpc":"2.0", "method": "eth_accounts", "params":[]}')
        ACCOUNT=$(echo "$DATA" | jq -r '.result[0]')
        echo "$(date '+%Y-%m-%d %H:%M:%S') - Deploying with $ACCOUNT"

        cd "$CONTRACTS_BEDROCK_DIR"

        # Send ETH to create2 deployer account, then deploy
        cast send --from "$ACCOUNT" \
            --rpc-url "$EPHEMERAL_GETH_URL" \
            --unlocked \
            --value '1ether' \
            0x3fAB184622Dc19b6109349B94811493BF2a45362 
        echo "publishing raw tx for create2 deployer"
        cast publish --rpc-url "$EPHEMERAL_GETH_URL" \
            '0xf8a58085174876e800830186a08080b853604580600e600039806000f350fe7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf31ba02222222222222222222222222222222222222222222222222222222222222222a02222222222222222222222222222222222222222222222222222222222222222' \

        echo "Deploying L1 contracts"
        forge script scripts/Deploy.s.sol:Deploy --sender $ACCOUNT --broadcast --rpc-url $EPHEMERAL_GETH_URL --unlocked

        # Copy .deploy artifact before sync
        cp $TMP_L1_DEPLOYMENTS_PATH $L1_DEPLOYMENTS_PATH 

        echo "Syncing L1 contracts"
        forge script scripts/Deploy.s.sol:Deploy --sig 'sync()' --broadcast --rpc-url $EPHEMERAL_GETH_URL

        # Send debug_dumpBlock request to geth, save res to allocs.json
        BODY='{"id":3, "jsonrpc":"2.0", "method": "debug_dumpBlock", "params":["latest"]}'
        curl -s -X POST \
            -H "Content-type: application/json" \
            -d "${BODY}" \
            "${EPHEMERAL_GETH_URL}" | jq -r '.result' > $ALLOCS_PATH

        # Kill ephemmeral geth in dev mode, we need to mutate l1 genesis and start again
        kill $GETH_PID
    else
        echo "allocs-l1.json already exist"
    fi

    # HACKY HACKY, no idea why replacing this timestamp field is needed.
    # See https://github.com/ethereum-optimism/optimism/blob/ee644a0bf55cae97e847aeecef07c059a2de3160/bedrock-devnet/devnet/__init__.py#L192
    jq --arg ts "$(printf '0x%x\n' $(date +%s))" '.l1GenesisBlockTimestamp = $ts' "$DEVNET_CONFIG_PATH" > temp.json && mv temp.json "$DEVNET_CONFIG_PATH"

    cd $OP_NODE_DIR
    # Create l1 genesis 
    go run cmd/main.go genesis l1 \
        --deploy-config $DEVNET_CONFIG_PATH \
        --l1-allocs $ALLOCS_PATH \
        --l1-deployments $L1_DEPLOYMENTS_PATH \
        --outfile.l1 $GENESIS_L1_PATH
else 
    echo "genesis-l1.json already exist"
fi    

# Signal L1 to start
touch /shared-optimism/start_l1
echo "signaled L1 to start"

# Wait for L1 to start
L1_URL="http://l1-geth:8545"
wait_on_rpc_server "$L1_URL"

if [ ! -e "$GENESIS_L2_PATH" ]; then
    echo "Generating genesis-l2.json and rollup.json"
    cd $OP_NODE_DIR
    go run cmd/main.go genesis l2 \
        --l1-rpc $L1_URL \
        --deploy-config $DEVNET_CONFIG_PATH \
        --deployment-dir $DEPLOYMENT_DIR \
        --outfile.l2 $GENESIS_L2_PATH \
        --outfile.rollup $ROLLUP_CONFIG_PATH
else 
    echo "genesis-l2.json and rollup.json already exist"
fi

touch /shared-optimism/start_l2
echo "signaled L2 to start"
echo "Coordintor setup complete"
exit 0
