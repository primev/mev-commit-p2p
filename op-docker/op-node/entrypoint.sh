#!/bin/bash
set -exu

L2_GETH_URL="http://op-geth:8545"

# Wait on op-geth to start before starting op-node
while ! curl -s -X POST "${L2_GETH_URL}" -H "Content-type: application/json" \
    -d '{"id":1, "jsonrpc":"2.0", "method": "eth_chainId", "params":[]}' | grep -q "jsonrpc"; do
    sleep 5 # sec
done

exec op-node \
    --l1=ws://l1-geth:8546 \
    --l2=http://op-geth:8551 \
    --l2.jwt-secret=/shared-optimism/ops-bedrock/test-jwt-secret.txt \
    --sequencer.enabled \
    --sequencer.l1-confs=0 \
    --verifier.l1-confs=0 \
    --p2p.sequencer.key=8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba \
    --rollup.config=/shared-optimism/.devnet/rollup.json \
    --rpc.addr=0.0.0.0 \
    --rpc.port=8545 \
    --p2p.listen.ip=0.0.0.0 \
    --p2p.listen.tcp=9003 \
    --p2p.listen.udp=9003 \
    --p2p.scoring.peers=light \
    --p2p.ban.peers=true \
    --p2p.priv.path=/shared-optimism/ops-bedrock/p2p-node-key.txt \
    --metrics.enabled \
    --metrics.addr=0.0.0.0 \
    --metrics.port=7300 \
    --pprof.enabled \
    --rpc.enable-admin 
