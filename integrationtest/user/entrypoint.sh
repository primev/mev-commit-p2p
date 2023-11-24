#!/bin/sh

sleep 30

echo "starting user-emulator with : ${USER_IP}"
/app/user-emulator --server-addr ${USER_IP} --rpc-addr ${RPC_URL}
