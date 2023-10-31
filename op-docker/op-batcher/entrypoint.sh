#!/bin/bash
set -exu

L1_GETH_URL="http://l1-geth:8545"
OP_GETH_URL="http://op-geth:8545"
OP_NODE_URL="http://op-node:8545"

# Wait on op-node to start before starting op-batcher
while ! curl -s -X POST "${OP_NODE_URL}" -H "Content-type: application/json" \
    -d '{"id":1, "jsonrpc":"2.0", "method": "eth_chainId", "params":[]}' | grep -q "jsonrpc"; do
    sleep 5 # sec
done

exec op-batcher \
    --l1-eth-rpc=$L1_GETH_URL \
    --l2-eth-rpc=$OP_GETH_URL \
    --rollup-rpc=$OP_NODE_URL \
    --max-channel-duration=1 \
    --sub-safety-margin=4 \
    --poll-interval=1s \
    --num-confirmations=1 \
    --mnemonic="test test test test test test test test test test test junk" \
    --sequencer-hd-path="m/44'/60'/0'/0/2" \
    --pprof.enabled \
    --metrics.enabled \
    --rpc.enable-admin
